package app

import (
	"io"
	"testing"
	"time"
)

func TestParseConfigFromMatrixFlags(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cfg, err := ParseConfig([]string{
		"-interfaces", "lo",
		"-targets", "127.0.0.1,8.8.8.8",
		"-interval", "3s",
		"-timeout", "2s",
	}, io.Discard)
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if got, want := len(cfg.Checks), 2; got != want {
		t.Fatalf("unexpected check count: got %d want %d", got, want)
	}
	if cfg.Interval != 3*time.Second {
		t.Fatalf("unexpected interval: %v", cfg.Interval)
	}
	if cfg.Timeout != 2*time.Second {
		t.Fatalf("unexpected timeout: %v", cfg.Timeout)
	}
}

func TestParseConfigFromExplicitChecks(t *testing.T) {
	cfg, err := ParseConfig([]string{
		"-check", "lo=127.0.0.1,127.0.0.2",
		"-check", "lo=127.0.0.2",
	}, io.Discard)
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if got, want := len(cfg.Checks), 2; got != want {
		t.Fatalf("unexpected deduped check count: got %d want %d", got, want)
	}
}

func TestParseConfigRequiresInputs(t *testing.T) {
	if _, err := ParseConfig(nil, io.Discard); err == nil {
		t.Fatal("expected error when no checks are provided")
	}
}
