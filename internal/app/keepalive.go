package app

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// defaultKeepalive is how often the server pings an idle WebSocket. Hosts
// and proxies close connections that stay quiet for about a minute.
const defaultKeepalive = 25 * time.Second

// listen keeps a WebSocket whose client only listens: it reads until the
// client goes away (so close frames and pongs are handled) and pings it
// every keepalive interval, closing it when a ping goes unanswered.
func (s *Server) listen(ctx context.Context, conn *websocket.Conn) {
	interval := s.keepalive
	if interval <= 0 {
		interval = defaultKeepalive
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				pingCtx, done := context.WithTimeout(ctx, interval)
				err := conn.Ping(pingCtx)
				done()
				if err != nil {
					_ = conn.Close(websocket.StatusGoingAway, "no answer to ping")
					return
				}
			}
		}
	}()
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}

// healthz answers the host's health checks: 200 while the database answers.
// Open to everyone, and tells nothing else.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}
