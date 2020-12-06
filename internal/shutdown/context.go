package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func New() (context.Context, func()) {
	return InterruptContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

func InterruptContext(ctx context.Context, signals ...os.Signal) (context.Context, func()) {
	ctx, closer := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)

	go func() {
		select {
		case <-c:
			closer()
		case <-ctx.Done():
		}
	}()

	return ctx, closer
}
