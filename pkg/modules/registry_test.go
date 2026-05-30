// LLM usage: this test file was generated with deepseek-v4-pro and modified manually.
package modules

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// --- test helpers ---

type testCfg struct {
	Name string
}

type testCfg2 struct {
	Value int
}

type testIface interface {
	IsTest() bool
}

func (c testCfg) IsTest() bool { return true }

type testModule struct {
	cfg any
}

func (m *testModule) Type() any {
	return m
}

func (m *testModule) ConfigMatches(cfg any) bool {
	return m.cfg == cfg
}

// newTestBuildFunc returns a BuildFunc that creates a *testModule holding cfg.
func newTestBuildFunc() BuildFunc {
	return func(_ context.Context, cfg any) (Module, error) {
		return &testModule{cfg: cfg}, nil
	}
}

// newFailingBuildFunc returns a BuildFunc that always fails.
func newFailingBuildFunc() BuildFunc {
	return func(_ context.Context, cfg any) (Module, error) {
		return nil, context.DeadlineExceeded
	}
}

// newCtxAssertBuildFunc returns a BuildFunc that errors if ctx is not the expected one.
func newCtxAssertBuildFunc(expectedCtx context.Context) BuildFunc {
	return func(ctx context.Context, cfg any) (Module, error) {
		if ctx != expectedCtx {
			return nil, context.Canceled
		}
		return &testModule{cfg: cfg}, nil
	}
}

// --- tests ---

func TestRegisterModule(t *testing.T) {
	// Reset registry for a clean state.
	buildFuncRegistry = map[reflect.Type]BuildFunc{}

	t.Run("register struct type", func(t *testing.T) {
		err := RegisterModule(testCfg{}, newTestBuildFunc())
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("register interface type via pointer", func(t *testing.T) {
		// Reset first; the previous sub-test registered testCfg.
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		err := RegisterModule((*testIface)(nil), newTestBuildFunc())
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		RegisterModule(testCfg{}, newTestBuildFunc())
		err := RegisterModule(testCfg{}, newTestBuildFunc())
		if err == nil {
			t.Fatal("expected error for duplicate registration")
		}
		if !strings.Contains(err.Error(), "already registered") {
			t.Fatalf("error should contain 'already registered', got: %v", err)
		}
	})

	t.Run("second distinct type ok", func(t *testing.T) {
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		RegisterModule(testCfg{}, newTestBuildFunc())
		err := RegisterModule(testCfg2{}, newTestBuildFunc())
		if err != nil {
			t.Fatalf("expected nil for distinct type, got %v", err)
		}
	})
}

func TestBuildModule(t *testing.T) {
	// Each sub-test resets the registry.
	ctx := context.Background()

	t.Run("exact struct match", func(t *testing.T) {
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		RegisterModule(testCfg{}, newTestBuildFunc())

		cfg := testCfg{Name: "hello"}
		m, err := BuildModule(ctx, cfg)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		tm, ok := m.(*testModule)
		if !ok {
			t.Fatalf("expected *testModule, got %T", m)
		}
		if !tm.ConfigMatches(cfg) {
			t.Fatalf("module cfg mismatch")
		}
	})

	t.Run("interface match", func(t *testing.T) {
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		RegisterModule((*testIface)(nil), newTestBuildFunc())

		cfg := testCfg{Name: "iface-match"}
		m, err := BuildModule(ctx, cfg)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		tm := m.(*testModule)
		if !tm.ConfigMatches(cfg) {
			t.Fatalf("module cfg mismatch")
		}
	})

	t.Run("no match", func(t *testing.T) {
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		RegisterModule(testCfg{}, newTestBuildFunc())

		_, err := BuildModule(ctx, testCfg2{Value: 42})
		if err == nil {
			t.Fatal("expected error for unmatched type")
		}
		if !strings.Contains(err.Error(), "no registered handler matches") {
			t.Fatalf("error should contain 'no registered handler matches', got: %v", err)
		}
	})

	t.Run("build func returns error", func(t *testing.T) {
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		RegisterModule(testCfg{}, newFailingBuildFunc())

		_, err := BuildModule(ctx, testCfg{Name: "fail"})
		if err == nil {
			t.Fatal("expected error from build func")
		}
	})

	t.Run("context propagated", func(t *testing.T) {
		buildFuncRegistry = map[reflect.Type]BuildFunc{}
		type ctxKey struct{}
		ctxWithVal := context.WithValue(ctx, ctxKey{}, "sentinel")
		RegisterModule(testCfg{}, newCtxAssertBuildFunc(ctxWithVal))

		_, err := BuildModule(ctxWithVal, testCfg{})
		if err != nil {
			t.Fatalf("expected nil error (context should match), got %v", err)
		}
	})
}
