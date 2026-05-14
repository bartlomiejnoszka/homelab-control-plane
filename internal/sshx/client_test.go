package sshx

import (
	"context"
	"errors"
	"testing"
)

type fakeRunner struct {
	output string
	err    error
}

func (f fakeRunner) Run(_ context.Context, _ string) (string, error) { return f.output, f.err }

func TestFakeRunnerSuccess(t *testing.T) {
	var r Runner = fakeRunner{output: "ok"}
	out, err := r.Run(context.Background(), "pct list")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out != "ok" {
		t.Fatalf("want ok, got %q", out)
	}
}

func TestFakeRunnerError(t *testing.T) {
	var r Runner = fakeRunner{err: errors.New("boom")}
	_, err := r.Run(context.Background(), "pct list")
	if err == nil {
		t.Fatal("expected error")
	}
}
