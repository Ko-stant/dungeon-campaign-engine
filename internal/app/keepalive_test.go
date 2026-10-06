package app

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// Hosts close WebSockets that stay quiet for a minute or so; the server
// pings each idle client well before that.
func TestStreamsPingIdleClients(t *testing.T) {
	srv := testServerWith(t, func(s *Server) { s.keepalive = 20 * time.Millisecond })
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	_, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "Night"})
	sess := decodeAny[SessionResponse](t, data)

	for _, stream := range []string{"stream", "player-stream"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pinged := make(chan struct{}, 1)
		conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/sessions/"+sess.ID+"/"+stream, &websocket.DialOptions{
			OnPingReceived: func(context.Context, []byte) bool {
				select {
				case pinged <- struct{}{}:
				default:
				}
				return true
			},
		})
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		conn.CloseRead(ctx) // handle control frames (pings) in the background
		select {
		case <-pinged:
		case <-ctx.Done():
			t.Errorf("%s: no ping from the server", stream)
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
		cancel()
	}
}

func TestHealthCheck(t *testing.T) {
	srv, _ := devServer(t)
	// Signed out, with sign-in on: the host's health check needs no account.
	resp, body := newBrowser(t, srv.URL).get("/healthz")
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(body) != "ok" {
		t.Errorf("health: %d %q", resp.StatusCode, body)
	}
}
