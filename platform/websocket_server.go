package platform

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketServer 웹소켓 서버
type WebSocketServer struct {
	upgrader  websocket.Upgrader
	clients   map[string]*Client
	clientsMu sync.RWMutex
	workerMgr *WorkerManager
	port      string
	server    *http.Server
}

// Client 웹소켓 클라이언트
type Client struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
	Server *WebSocketServer
	mu     sync.Mutex
}

// NewWebSocketServer 새로운 웹소켓 서버를 생성합니다
func NewWebSocketServer(workerMgr *WorkerManager, port string) *WebSocketServer {
	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // CORS 허용
			},
		},
		clients:   make(map[string]*Client),
		workerMgr: workerMgr,
		port:      port,
	}
}

// Start 웹소켓 서버를 시작합니다
func (ws *WebSocketServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.handleWebSocket)

	ws.server = &http.Server{
		Addr:    ":" + ws.port,
		Handler: mux,
	}

	return ws.server.ListenAndServe()
}

// Stop 웹소켓 서버를 중지합니다
func (ws *WebSocketServer) Stop() error {
	ws.clientsMu.Lock()
	for _, client := range ws.clients {
		client.close()
	}
	ws.clientsMu.Unlock()

	if ws.server != nil {
		return ws.server.Close()
	}
	return nil
}

// handleWebSocket 웹소켓 연결을 처리합니다
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("웹소켓 업그레이드 실패: %v", err)
		return
	}

	// 클라이언트 ID 생성
	clientID := generateClientID()
	userID := r.URL.Query().Get("userId")

	if userID == "" {
		log.Printf("사용자 ID가 없습니다")
		conn.Close()
		return
	}

	client := &Client{
		ID:     clientID,
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Server: ws,
	}

	ws.clientsMu.Lock()
	ws.clients[clientID] = client
	ws.clientsMu.Unlock()

	// 클라이언트 고루틴 시작
	go client.readPump()
	go client.writePump()

	// 워커 매니저에 클라이언트 등록
	ws.workerMgr.RegisterWebSocketClient(userID, client)
}

// generateClientID 클라이언트 ID를 생성합니다
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 랜덤 문자열을 생성합니다
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// readPump 클라이언트로부터 메시지를 읽습니다
func (c *Client) readPump() {
	defer func() {
		c.Server.unregisterClient(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("웹소켓 읽기 오류: %v", err)
			}
			break
		}

		// 메시지 처리 (개발모드 노이즈 제거)
		if string(message) == "runtime:ready" {
			continue
		}
		log.Printf("클라이언트 %s로부터 메시지 수신: %s", c.ID, string(message))
	}
}

// writePump 클라이언트에게 메시지를 씁니다
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage 클라이언트에게 메시지를 전송합니다
func (c *Client) SendMessage(message []byte) {
	select {
	case c.Send <- message:
	default:
		c.close()
	}
}

// close 클라이언트를 닫습니다
func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn != nil {
		c.Conn.Close()
	}
	close(c.Send)
}

// unregisterClient 클라이언트를 등록 해제합니다
func (ws *WebSocketServer) unregisterClient(client *Client) {
	ws.clientsMu.Lock()
	delete(ws.clients, client.ID)
	ws.clientsMu.Unlock()

	// 워커 매니저에서 클라이언트 제거
	ws.workerMgr.UnregisterWebSocketClient(client.UserID, client)

}

// BroadcastToUser 특정 사용자에게 메시지를 브로드캐스트합니다
func (ws *WebSocketServer) BroadcastToUser(userID string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("메시지 마샬링 실패: %v", err)
		return
	}

	ws.clientsMu.RLock()
	defer ws.clientsMu.RUnlock()

	for _, client := range ws.clients {
		if client.UserID == userID {
			client.SendMessage(data)
		}
	}
}

// GetClientCount 클라이언트 수를 반환합니다
func (ws *WebSocketServer) GetClientCount() int {
	ws.clientsMu.RLock()
	defer ws.clientsMu.RUnlock()
	return len(ws.clients)
}
