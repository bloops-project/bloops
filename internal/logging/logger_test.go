package logging

import (
	"context"
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	logger := NewLogger(true)
	if logger == nil {
		t.Fatal("logger cannot be nil")
	}
}

func TestDefaultLogger(t *testing.T) {
	t.Parallel()

	logger1 := DefaultLogger()
	if logger1 == nil {
		t.Fatal("logger cannot be nil")
	}

	logger2 := DefaultLogger()
	if logger2 == nil {
		t.Fatal("logger cannot be nil")
	}

	if logger1 != logger2 {
		t.Errorf("expected %#v got %#v", logger1, logger2)
	}
}

func TestContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger1 := FromContext(ctx)
	if logger1 == nil {
		t.Fatal("logger cannot be nil")
	}

	ctx = WithLogger(ctx, logger1)

	logger2 := FromContext(ctx)
	if logger1 != logger2 {
		t.Errorf("expected %#v got %#v", logger1, logger2)
	}
}
