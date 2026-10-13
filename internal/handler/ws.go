package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/platform/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = (wsPongWait * 9) / 10
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type wsClient struct {
	hub  *wsHub
	conn *websocket.Conn
	send chan []byte
}

type wsHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
}

var wsHubInstance = &wsHub{clients: map[*wsClient]bool{}}

func (h *wsHub) register(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *wsHub) unregister(c *wsClient) {
	h.mu.Lock()
	if h.clients[c] {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

func (h *wsHub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
		}
	}
}

func (h *wsHub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

type wsEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func broadcastWS(eventType string, data interface{}) {
	if wsHubInstance.count() == 0 {
		return
	}
	msg, err := json.Marshal(wsEvent{Type: eventType, Data: data})
	if err != nil {
		return
	}
	wsHubInstance.broadcast(msg)
}

func broadcastRoomEvent(action string, roomID int) {
	var room models.PenginapanRoom
	if err := db.PenginapanDB.Preload("Location").First(&room, roomID).Error; err != nil {
		return
	}
	broadcastWS("room", map[string]interface{}{"action": action, "room": room})
}

func broadcastRoomDeleted(roomID int) {
	broadcastWS("room", map[string]interface{}{"action": "delete", "room_id": roomID})
}

func broadcastGuestEvent(action string, guestIDs []int) {
	broadcastWS("guest", map[string]interface{}{"action": action, "guest_ids": guestIDs})
}

func HandleWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": false, "message": "token wajib di query ?token="})
		return
	}
	if _, err := auth.ValidateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": false, "message": "token tidak valid"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("ws upgrade failed: %v", err)
		return
	}

	client := &wsClient{hub: wsHubInstance, conn: conn, send: make(chan []byte, 64)}
	wsHubInstance.register(client)

	go client.writePump()
	go client.readPump()

	logger.Info("WS client connected (total=%d)", wsHubInstance.count())
}

func (c *wsClient) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
		logger.Info("WS client disconnected (total=%d)", c.hub.count())
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
