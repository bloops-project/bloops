package server

import (
	"context"
	"fmt"
	"net/http"
)

func HandleHealth(_ context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"status": "ok"}`)
	})
}
