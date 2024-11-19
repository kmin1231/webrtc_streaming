package main

import (
    "log"
    "net/http"
    "sync"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

type SignalingServer struct {
    mutex          sync.Mutex
    broadcasterConn *websocket.Conn
    viewerConn     *websocket.Conn
}

func NewSignalingServer() *SignalingServer {
    return &SignalingServer{}
}

func (s *SignalingServer) handleConnection(conn *websocket.Conn) {
    // 역할 구분 메시지 수신
    _, message, err := conn.ReadMessage()
    if err != nil {
        log.Printf("Failed to read initial message: %v", err)
        return
    }
    role := string(message)
    log.Printf("New connection with role: %s", role)

    s.mutex.Lock()
    switch role {
    case "broadcaster":
        if s.broadcasterConn != nil {
            log.Println("Broadcaster already exists, rejecting new connection")
            conn.WriteMessage(websocket.TextMessage, []byte("error: broadcaster already exists"))
            s.mutex.Unlock()
            return
        }
        s.broadcasterConn = conn
        log.Println("Broadcaster connected")
        
    case "viewer":
        if s.viewerConn != nil {
            log.Println("Viewer already exists, rejecting new connection")
            conn.WriteMessage(websocket.TextMessage, []byte("error: viewer already exists"))
            s.mutex.Unlock()
            return
        }
        s.viewerConn = conn
        log.Println("Viewer connected")
        
    default:
        log.Printf("Unknown role: %s", role)
        conn.WriteMessage(websocket.TextMessage, []byte("error: unknown role"))
        s.mutex.Unlock()
        return
    }
    s.mutex.Unlock()

    // 연결이 끊어졌을 때 정리
    defer func() {
        s.mutex.Lock()
        if conn == s.broadcasterConn {
            log.Println("Broadcaster disconnected")
            s.broadcasterConn = nil
        } else if conn == s.viewerConn {
            log.Println("Viewer disconnected")
            s.viewerConn = nil
        }
        s.mutex.Unlock()
        conn.Close()
    }()

    // 메시지 전달 루프
    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            log.Printf("Read error: %v", err)
            return
        }

        // JSON 메시지 종류 로깅
        log.Printf("Received message from %s: %s", role, string(message[:min(len(message), 100)]))

        s.mutex.Lock()
        switch {
        case conn == s.broadcasterConn && s.viewerConn != nil:
            log.Println("Forwarding message from broadcaster to viewer")
            if err := s.viewerConn.WriteMessage(messageType, message); err != nil {
                log.Printf("Error sending to viewer: %v", err)
                s.mutex.Unlock()
                return
            }
            
        case conn == s.viewerConn && s.broadcasterConn != nil:
            log.Println("Forwarding message from viewer to broadcaster")
            if err := s.broadcasterConn.WriteMessage(messageType, message); err != nil {
                log.Printf("Error sending to broadcaster: %v", err)
                s.mutex.Unlock()
                return
            }
        }
        s.mutex.Unlock()
    }
}

func (s *SignalingServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("Upgrade error: %v", err)
        return
    }
    
    go s.handleConnection(conn)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func main() {
    server := NewSignalingServer()
    http.Handle("/signal", server)
    
    addr := ":8080"
    log.Printf("Signaling server starting on %s", addr)
    log.Fatal(http.ListenAndServe(addr, nil))
}