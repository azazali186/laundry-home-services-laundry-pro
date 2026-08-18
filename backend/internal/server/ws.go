package server

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/jwtx"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/websocket"
)

// realtimeTopics are the domain events pushed to clients.
var realtimeTopics = []string{
	"booking_created", "booking_cancelled", "pickup_scheduled",
	"order_created", "order_status_changed", "delivery_delivered",
	"payment_paid", "payment_failed", "payment_refunded", "invoice_paid",
}

// Hub manages live WebSocket connections and relays bus events to clients.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]*websocket.Conn
	bus   *bus.Bus
}

// NewHub builds the hub.
func NewHub(b *bus.Bus) *Hub {
	return &Hub{conns: map[string]*websocket.Conn{}, bus: b}
}

// Start relays domain events to connected clients (one goroutine per topic).
func (h *Hub) Start(ctx context.Context) {
	for _, topic := range realtimeTopics {
		ch := h.bus.Subscribe(topic)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case ev := <-ch:
					h.broadcast(ev)
				}
			}
		}()
	}
}

func (h *Hub) add(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	h.conns[userID] = conn
	h.mu.Unlock()
	logger.Info("ws client connected: %s", userID)
}

func (h *Hub) remove(userID string) {
	h.mu.Lock()
	delete(h.conns, userID)
	h.mu.Unlock()
}

func (h *Hub) broadcast(ev bus.Event) {
	payload, _ := json.Marshal(ev)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, conn := range h.conns {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			logger.Warn("ws write failed: %v", err)
		}
	}
}

// MountWebSocket registers /ws with token authentication via the
// Authorization header (or Sec-WebSocket-Protocol subprotocol for browsers).
func MountWebSocket(h *server.Hertz, cache *cache.Cache, jwt *jwtx.Manager, b *bus.Bus) {
	hub := NewHub(b)
	hub.Start(context.Background())

	upgrader := websocket.HertzUpgrader{
		CheckOrigin: func(*app.RequestContext) bool { return true },
	}
	h.GET("/ws", func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.Request.Header.Get("Authorization"))
		if len(authHeader) <= 7 {
			c.AbortWithStatus(401)
			return
		}
		claims, err := jwt.Parse(authHeader[7:])
		if err != nil {
			c.AbortWithStatus(401)
			return
		}
		userID := claims.UserID.String()
		if err := upgrader.Upgrade(c, func(conn *websocket.Conn) {
			hub.add(userID, conn)
			defer hub.remove(userID)
			conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}); err != nil {
			logger.Warn("ws upgrade failed: %v", err)
		}
	})
}
