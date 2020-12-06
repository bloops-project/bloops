package shutdown

import (
	"context"
	"syscall"
	"testing"
)

func TestInterruptContext(t *testing.T) {
	ctx, cancel := InterruptContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	cancel()
	<-ctx.Done()
}
