package logging

import (
	"context"
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()
	logger := NewLogger(true)
	if logger == nil {
		t.Fatal("expected logger to never be nil")
	}
}

func TestDefaultLogger(t *testing.T) {
	t.Parallel()

	logger1 := DefaultLogger()
	if logger1 == nil {
		t.Fatal("expected logger to never be nil")
	}

	logger2 := DefaultLogger()
	if logger2 == nil {
		t.Fatal("expected logger to never be nil")
	}

	if logger1 != logger2 {
		t.Errorf("expected %#v to be %#v", logger1, logger2)
	}
}

func TestContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger1 := FromContext(ctx)
	if logger1 == nil {
		t.Fatal("expected logger to never be nil")
	}

	ctx = WithLogger(ctx, logger1)

	logger2 := FromContext(ctx)
	if logger1 != logger2 {
		t.Errorf("expected %#v to be %#v", logger1, logger2)
	}
}
