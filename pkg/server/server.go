package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"

	"github.com/wamphlett/remote-sensors-server/config"
	"github.com/wamphlett/remote-sensors-server/pkg/manager"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Server defines the methods required to run the HTTP server
type Server struct {
	httpServer *http.Server
}

// New creates a new fully configured Server
func New(cfg *config.Server, deviceManager *manager.Manager) *Server {
	router := httprouter.New()
	router.GET("/device/:deviceid", deviceHandler(deviceManager))

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	return &Server{
		httpServer: s,
	}
}

// Start starts the Server on the specified port
func (s *Server) Start() {
	go func() {
		if err := s.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
}

// Shutdown immediately closes any open connections
func (s *Server) Shutdown() {
	s.httpServer.Close()
}

// deviceHandler build the function to handle incoming websocket connections
func deviceHandler(m *manager.Manager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }

		// upgrade this connection to a WebSocket
		// connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		deviceID := params.ByName("deviceid")
		updates := m.Subscribe(deviceID)
		defer func() {
			log.Printf("client unsubscribed from device: %s\n", deviceID)
			m.Unsubscribe(deviceID, updates)
		}()

		log.Printf("client subscribed to device: %s\n", deviceID)

		closer := make(chan bool)

		go func() {
			for {
				select {
				case u := <-updates:
					if err := conn.WriteMessage(1, u); err != nil {
						log.Println(err)
						return
					}
				case <-closer:
					return
				}

			}
		}()

		// continuously read messages from the the client. all messages are
		// ignored except for close messages
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				closer <- true
				conn.Close()
				return
			}
		}
	}
}
