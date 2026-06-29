// LLM usage: generated with deepseek-v4-pro and modified manually
package manager

import (
	"context"
	"fmt"
	"testing"

	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/nodes/factory"
	"github.com/HT4w5/l4rt/pkg/nodes/node"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

type mockConfigCommon struct{}

func (c *mockConfigCommon) Log() log.Config {
	return nil
}

// mockHandlerConfig yields a simple handler (neither active nor dispatcher).
type mockHandlerConfig struct {
	mockConfigCommon
	tag string
}

func (c *mockHandlerConfig) Tag() string { return c.tag }

// mockActiveHandlerConfig yields an active handler.
type mockActiveHandlerConfig struct {
	mockConfigCommon
	tag string
}

func (c *mockActiveHandlerConfig) Tag() string { return c.tag }

// mockDispatcherConfig yields a dispatcher that depends on handlers identified
// by depTags.
type mockDispatcherConfig struct {
	mockConfigCommon
	tag     string
	depTags []string
}

func (c *mockDispatcherConfig) Tag() string { return c.tag }

// mockHandler is a bare handler that is neither active nor a dispatcher.
type mockHandler struct {
	tag string
}

func (h *mockHandler) Tag() string    { return h.tag }
func (h *mockHandler) String() string { return h.tag }

// mockActiveHandler implements ActiveHandler.
type mockActiveHandler struct {
	tag            string
	startCalled    bool
	shutdownCalled bool
}

func (h *mockActiveHandler) Tag() string                    { return h.tag }
func (h *mockActiveHandler) String() string                 { return h.tag }
func (h *mockActiveHandler) Start(context.Context) error    { h.startCalled = true; return nil }
func (h *mockActiveHandler) Shutdown(context.Context) error { h.shutdownCalled = true; return nil }

// mockDispatcher implements both Dispatcher and ActiveHandler. It depends on
// the handlers listed in depTags.
type mockDispatcher struct {
	tag      string
	depTags  []string
	injected map[string]node.Node
}

func (d *mockDispatcher) Tag() string    { return d.tag }
func (d *mockDispatcher) String() string { return d.tag }

func (d *mockDispatcher) Start(context.Context) error    { return nil }
func (d *mockDispatcher) Shutdown(context.Context) error { return nil }

func (d *mockDispatcher) InjectHandlers(getter func(string) (node.Node, error)) error {
	for _, depTag := range d.depTags {
		h, err := getter(depTag)
		if err != nil {
			return err
		}
		d.injected[depTag] = h
	}
	return nil
}

func (d *mockDispatcher) Handlers() []node.Node {
	out := make([]node.Node, 0, len(d.depTags))
	for _, tag := range d.depTags {
		out = append(out, d.injected[tag])
	}
	return out
}

// mockConfig implements manager.Config.
type mockConfig struct {
	handlerCfgs []node.Config
}

func (c *mockConfig) Nodes() []node.Config { return c.handlerCfgs }

// mockLoggerGetter implements log.Getter.
type mockLoggerGetter struct{}

func (m *mockLoggerGetter) GetLogger(cfg log.Config) (zerolog.Logger, error) {
	return zerolog.Nop(), nil
}

// ---------------------------------------------------------------------------
// Factory fallback: teach factory.NewHandler how to build our mock types.
// ---------------------------------------------------------------------------

func init() {
	factory.SetFallback(func(cfg node.Config) (node.Node, error) {
		switch v := cfg.(type) {
		case *mockHandlerConfig:
			return &mockHandler{tag: v.tag}, nil
		case *mockActiveHandlerConfig:
			return &mockActiveHandler{tag: v.tag}, nil
		case *mockDispatcherConfig:
			return &mockDispatcher{
				tag:      v.tag,
				depTags:  v.depTags,
				injected: make(map[string]node.Node),
			}, nil
		default:
			return nil, fmt.Errorf("unknown mock config type: %T", cfg)
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustNewManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	mgr, err := NewManager(cfg, zerolog.Logger{}, &mockLoggerGetter{})
	if err != nil {
		t.Fatalf("NewManager: unexpected error: %v", err)
	}
	return mgr
}

func startupTags(mgr *Manager) []string {
	tags := make([]string, 0, len(mgr.nodes.startupOrder))
	for _, h := range mgr.nodes.startupOrder {
		tags = append(tags, h.Tag())
	}
	return tags
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewManager_Empty(t *testing.T) {
	t.Parallel()
	cfg := &mockConfig{}
	mgr, err := NewManager(cfg, zerolog.Logger{}, &mockLoggerGetter{})
	if err != nil {
		t.Fatalf("expected no error for empty config, got: %v", err)
	}
	if len(mgr.nodes.startupOrder) != 0 {
		t.Fatalf("expected empty startupOrder, got %v", startupTags(mgr))
	}
}

func TestNewManager_SingleActiveHandler(t *testing.T) {
	t.Parallel()
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockActiveHandlerConfig{tag: "alpha"},
		},
	}
	mgr := mustNewManager(t, cfg)
	if tags := startupTags(mgr); len(tags) != 1 || tags[0] != "alpha" {
		t.Fatalf("expected startupOrder [alpha], got %v", tags)
	}
}

func TestNewManager_SimpleHandlerSkipped(t *testing.T) {
	t.Parallel()
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockHandlerConfig{tag: "simple"},
		},
	}
	mgr := mustNewManager(t, cfg)
	if tags := startupTags(mgr); len(tags) != 0 {
		t.Fatalf("expected empty startupOrder for non-active handler, got %v", tags)
	}
}

func TestNewManager_ActiveAndSimpleMixed(t *testing.T) {
	t.Parallel()
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockHandlerConfig{tag: "simple"},
			&mockActiveHandlerConfig{tag: "active"},
		},
	}
	mgr := mustNewManager(t, cfg)
	if tags := startupTags(mgr); len(tags) != 1 || tags[0] != "active" {
		t.Fatalf("expected startupOrder [active], got %v", tags)
	}
}

func TestNewManager_DispatcherOrdering(t *testing.T) {
	t.Parallel()
	// "alpha" is an active handler; "bravo" is a dispatcher that depends on alpha.
	// Expected startup order: [alpha, bravo]
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockActiveHandlerConfig{tag: "alpha"},
			&mockDispatcherConfig{tag: "bravo", depTags: []string{"alpha"}},
		},
	}
	mgr := mustNewManager(t, cfg)
	if tags := startupTags(mgr); len(tags) != 2 || tags[0] != "alpha" || tags[1] != "bravo" {
		t.Fatalf("expected startupOrder [alpha, bravo], got %v", tags)
	}
}

func TestNewManager_MultiLevelDeps(t *testing.T) {
	t.Parallel()
	// A -> B -> C chain.
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockActiveHandlerConfig{tag: "a"},
			&mockDispatcherConfig{tag: "b", depTags: []string{"a"}},
			&mockDispatcherConfig{tag: "c", depTags: []string{"b"}},
		},
	}
	mgr := mustNewManager(t, cfg)
	if tags := startupTags(mgr); len(tags) != 3 || tags[0] != "a" || tags[1] != "b" || tags[2] != "c" {
		t.Fatalf("expected startupOrder [a, b, c], got %v", tags)
	}
}

func TestNewManager_DuplicateTags(t *testing.T) {
	t.Parallel()
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockActiveHandlerConfig{tag: "dup"},
			&mockActiveHandlerConfig{tag: "dup"},
		},
	}
	_, err := NewManager(cfg, zerolog.Logger{}, &mockLoggerGetter{})
	if err == nil {
		t.Fatal("expected error for duplicate tags, got nil")
	}
}

func TestNewManager_CycleDetection(t *testing.T) {
	t.Parallel()
	// alpha -> bravo -> alpha -> ...
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockDispatcherConfig{tag: "alpha", depTags: []string{"bravo"}},
			&mockDispatcherConfig{tag: "bravo", depTags: []string{"alpha"}},
		},
	}
	_, err := NewManager(cfg, zerolog.Logger{}, &mockLoggerGetter{})
	if err == nil {
		t.Fatal("expected error for cyclic dependency, got nil")
	}
}

func TestNewManager_DispatcherInjection(t *testing.T) {
	t.Parallel()
	// Create a handler and a dispatcher that depends on it. After NewManager
	// returns, the dispatcher should have the handler injected.
	cfg := &mockConfig{
		handlerCfgs: []node.Config{
			&mockActiveHandlerConfig{tag: "dep"},
			&mockDispatcherConfig{tag: "disp", depTags: []string{"dep"}},
		},
	}
	mgr := mustNewManager(t, cfg)

	// Find the dispatcher and verify it has its dependency injected.
	disp, ok := mgr.nodes.all["disp"].(*mockDispatcher)
	if !ok {
		t.Fatal("expected disp to be *mockDispatcher")
	}
	if len(disp.injected) != 1 {
		t.Fatalf("expected 1 injected handler, got %d", len(disp.injected))
	}
	if h, ok := disp.injected["dep"]; !ok {
		t.Fatal("expected dep to be injected into disp")
	} else if h.Tag() != "dep" {
		t.Fatalf("expected injected handler tag 'dep', got %q", h.Tag())
	}
}
