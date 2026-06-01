// LLM usage: this test file was created with deepseek-v4-pro and modified manually.
package log

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// ──────────────────────────────────────────────────────────────
// Mock configs
// ──────────────────────────────────────────────────────────────

// mockFactoryConfig implements FactoryConfig for testing.
type mockFactoryConfig struct {
	pullInterval time.Duration
	bufferSize   int
	timeFormat   string
}

func (m mockFactoryConfig) PullInterval() time.Duration { return m.pullInterval }
func (m mockFactoryConfig) BufferSize() int             { return m.bufferSize }
func (m mockFactoryConfig) TimeFormat() string          { return m.timeFormat }

// newMockFactoryConfig returns a FactoryConfig tuned for fast-flushing tests.
func newMockFactoryConfig() mockFactoryConfig {
	return mockFactoryConfig{
		pullInterval: 1 * time.Millisecond,
		bufferSize:   10,
		timeFormat:   zerolog.TimeFormatUnix,
	}
}

// mockConfig implements Config for testing.
type mockConfig struct {
	level        zerolog.Level
	output       string
	addCaller    bool
	addTimestamp bool
}

func (m mockConfig) Level() zerolog.Level { return m.level }
func (m mockConfig) Output() string       { return m.output }
func (m mockConfig) AddCaller() bool      { return m.addCaller }
func (m mockConfig) AddTimestamp() bool   { return m.addTimestamp }

// newMockConfig returns a Config with the given level and output, plus sensible defaults.
func newMockConfig(level zerolog.Level, output string) mockConfig {
	return mockConfig{
		level:        level,
		output:       output,
		addTimestamp: true,
	}
}

// ──────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────

// captureStdoutStderr replaces the global stdout/stderr variables with pipe
// writers and returns the read-ends plus a restore function.
func captureStdoutStderr() (stdoutR, stderrR *os.File, restore func()) {
	origStdout, origStderr := stdout, stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	stdout = wOut
	stderr = wErr

	return rOut, rErr, func() {
		stdout = origStdout
		stderr = origStderr
		wOut.Close()
		wErr.Close()
	}
}

// readPipe reads all available data from a pipe reader.
func readPipe(r *os.File) string {
	var buf bytes.Buffer
	// Close the write-end first so Read doesn't block forever.
	// The caller must ensure the write-end is closed.
	io.Copy(&buf, r)
	return buf.String()
}

// ──────────────────────────────────────────────────────────────
// 1. stdout / stderr mocking
// ──────────────────────────────────────────────────────────────

func TestFactory_GetLogger_Stdout(t *testing.T) {
	stdoutR, stderrR, restore := captureStdoutStderr()
	defer restore()

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)
	defer cleanup()

	logCfg := newMockConfig(zerolog.InfoLevel, "stdout")
	logger, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		t.Fatalf("GetLogger(stdout): %v", err)
	}

	logger.Info().Msg("hello stdout")

	// Close write-ends so pipes unblock.
	restore()

	stdoutOut := readPipe(stdoutR)
	stderrOut := readPipe(stderrR)

	if !strings.Contains(stdoutOut, "hello stdout") {
		t.Errorf("stdout should contain 'hello stdout', got: %q", stdoutOut)
	}
	if stderrOut != "" {
		t.Errorf("stderr should be empty, got: %q", stderrOut)
	}
}

func TestFactory_GetLogger_Stderr(t *testing.T) {
	stdoutR, stderrR, restore := captureStdoutStderr()
	defer restore()

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)
	defer cleanup()

	logCfg := newMockConfig(zerolog.InfoLevel, "stderr")
	logger, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		t.Fatalf("GetLogger(stderr): %v", err)
	}

	logger.Info().Msg("hello stderr")

	restore()

	stdoutOut := readPipe(stdoutR)
	stderrOut := readPipe(stderrR)

	if !strings.Contains(stderrOut, "hello stderr") {
		t.Errorf("stderr should contain 'hello stderr', got: %q", stderrOut)
	}
	if stdoutOut != "" {
		t.Errorf("stdout should be empty, got: %q", stdoutOut)
	}
}

func TestFactory_GetLogger_EmptyOutput_FallsBackToStderr(t *testing.T) {
	stdoutR, stderrR, restore := captureStdoutStderr()
	defer restore()

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)
	defer cleanup()

	logCfg := newMockConfig(zerolog.InfoLevel, "") // empty → stderr
	logger, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		t.Fatalf("GetLogger(\"\"): %v", err)
	}

	logger.Info().Msg("fallback to stderr")

	restore()

	stdoutOut := readPipe(stdoutR)
	stderrOut := readPipe(stderrR)

	if !strings.Contains(stderrOut, "fallback to stderr") {
		t.Errorf("stderr should contain 'fallback to stderr', got: %q", stderrOut)
	}
	if stdoutOut != "" {
		t.Errorf("stdout should be empty, got: %q", stdoutOut)
	}
}

// ──────────────────────────────────────────────────────────────
// 2. Writing to a temp file
// ──────────────────────────────────────────────────────────────

func TestFactory_GetLogger_TempFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test.log"

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)

	logCfg := newMockConfig(zerolog.InfoLevel, tmpFile)
	logger, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		cleanup()
		t.Fatalf("GetLogger(file): %v", err)
	}

	logger.Info().Msg("file log line 1")
	logger.Info().Msg("file log line 2")

	// Cleanup flushes the diode writer before closing files.
	cleanup()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "file log line 1") {
		t.Errorf("file should contain 'file log line 1', got: %q", content)
	}
	if !strings.Contains(content, "file log line 2") {
		t.Errorf("file should contain 'file log line 2', got: %q", content)
	}
}

func TestFactory_GetLogger_OutputCaching(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/cached.log"

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)

	logCfg := newMockConfig(zerolog.InfoLevel, tmpFile)

	// First call creates the writer and opens the file.
	logger1, err := factory.GetLogger(logCfg, "mod-a")
	if err != nil {
		cleanup()
		t.Fatalf("GetLogger first call: %v", err)
	}
	logger1.Info().Msg("from mod-a")

	// Second call with same output reuses the cached writer.
	logger2, err := factory.GetLogger(logCfg, "mod-b")
	if err != nil {
		cleanup()
		t.Fatalf("GetLogger second call: %v", err)
	}
	logger2.Info().Msg("from mod-b")

	cleanup()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "from mod-a") {
		t.Errorf("file should contain 'from mod-a', got: %q", content)
	}
	if !strings.Contains(content, "from mod-b") {
		t.Errorf("file should contain 'from mod-b', got: %q", content)
	}
}

// ──────────────────────────────────────────────────────────────
// 3. Multiple logger concurrency
// ──────────────────────────────────────────────────────────────

func TestFactory_GetLogger_Concurrent(t *testing.T) {
	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)
	defer cleanup()

	const (
		numGoroutines  = 20
		msgsPerRoutine = 5
	)

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			logCfg := newMockConfig(zerolog.InfoLevel, "stdout")
			logger, err := factory.GetLogger(logCfg, "concurrent")
			if err != nil {
				errors <- err
				return
			}

			for j := 0; j < msgsPerRoutine; j++ {
				logger.Info().Int("goroutine", id).Int("msg", j).Msg("concurrent log")
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("GetLogger error: %v", err)
	}
}

func TestFactory_GetLogger_Concurrent_MixedOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := tmpDir + "/a.log"
	fileB := tmpDir + "/b.log"

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)

	const numGoroutines = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*2)

	// Goroutines writing to file A
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logCfg := newMockConfig(zerolog.InfoLevel, fileA)
			logger, err := factory.GetLogger(logCfg, "mod-a")
			if err != nil {
				errors <- err
				return
			}
			logger.Info().Int("goroutine", id).Msg("in file A")
		}(i)
	}

	// Goroutines writing to file B
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logCfg := newMockConfig(zerolog.InfoLevel, fileB)
			logger, err := factory.GetLogger(logCfg, "mod-b")
			if err != nil {
				errors <- err
				return
			}
			logger.Info().Int("goroutine", id).Msg("in file B")
		}(i)
	}

	wg.Wait()
	cleanup()
	close(errors)

	for err := range errors {
		t.Errorf("GetLogger error: %v", err)
	}

	// Verify both files have content.
	for _, fp := range []string{fileA, fileB} {
		data, err := os.ReadFile(fp)
		if err != nil {
			t.Errorf("ReadFile(%s): %v", fp, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("file %s is empty, expected log lines", fp)
		}
	}
}

// ──────────────────────────────────────────────────────────────
// 4. Edge cases
// ──────────────────────────────────────────────────────────────

func TestFactory_GetLogger_DisabledLevel_UsesDiscard(t *testing.T) {
	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)
	defer cleanup()

	logCfg := newMockConfig(zerolog.Disabled, "stdout")
	logger, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		t.Fatalf("GetLogger(disabled): %v", err)
	}

	// Logging with Disabled level should not panic and produce no output.
	// We verify by writing and making sure nothing crashes.
	logger.Info().Msg("this should be discarded")
	logger.Error().Msg("this too")
}

func TestFactory_GetLogger_FileOpenError(t *testing.T) {
	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)
	defer cleanup()

	// A path with a non-existent intermediate directory.
	logCfg := newMockConfig(zerolog.InfoLevel, "/nonexistent/path/should/fail.log")
	_, err := factory.GetLogger(logCfg, "test-module")
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestFactory_GetLogger_AddCaller(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/caller.log"

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)

	logCfg := mockConfig{
		level:        zerolog.InfoLevel,
		output:       tmpFile,
		addCaller:    true,
		addTimestamp: false,
	}
	logger, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		cleanup()
		t.Fatalf("GetLogger(addCaller): %v", err)
	}

	logger.Info().Msg("with caller")

	cleanup()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "factory_test.go") {
		t.Errorf("expected caller file in log, got: %q", content)
	}
}

func TestFactory_GetLogger_NoTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/nots.log"

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)

	logCfg := mockConfig{
		level:        zerolog.InfoLevel,
		output:       tmpFile,
		addTimestamp: false,
	}
	logger, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		cleanup()
		t.Fatalf("GetLogger(no timestamp): %v", err)
	}

	logger.Info().Msg("no timestamp")

	cleanup()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	// Since TimeFormatUnix is set on the package, a timestamp would show digits.
	// With addTimestamp=false, the message should still appear.
	if !strings.Contains(content, "no timestamp") {
		t.Errorf("expected 'no timestamp' in log, got: %q", content)
	}
}

func TestFactory_NewFactory_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/cleanup.log"

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)

	logCfg := newMockConfig(zerolog.InfoLevel, tmpFile)
	_, err := factory.GetLogger(logCfg, "test-module")
	if err != nil {
		cleanup()
		t.Fatalf("GetLogger: %v", err)
	}

	// Cleanup should close files without panicking.
	cleanup()

	// Second cleanup should also be safe (no-op on already-closed).
	cleanup()
}

func TestFactory_GetLogger_ModuleField(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/module.log"

	factoryCfg := newMockFactoryConfig()
	factory, cleanup := NewFactory(factoryCfg)

	logCfg := newMockConfig(zerolog.InfoLevel, tmpFile)
	logger, err := factory.GetLogger(logCfg, "my-module")
	if err != nil {
		cleanup()
		t.Fatalf("GetLogger: %v", err)
	}

	logger.Info().Msg("module test")

	cleanup()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"module":"my-module"`) {
		t.Errorf("expected module field in log, got: %q", content)
	}
}
