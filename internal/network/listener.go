package network

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"p2p-messenger/internal/crypto"
	"p2p-messenger/internal/entity"
	"p2p-messenger/internal/proto"
)

var (
	upgrader = websocket.Upgrader{}
)

type Listener struct {
	proto *proto.Proto
	addr  string
}

func NewListener(addr string, proto *proto.Proto) *Listener {
	return &Listener{
		proto: proto,
		addr:  addr,
	}
}

func (l *Listener) chat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Establish Noise Protocol session as responder
	session, err := crypto.NewResponderSession(l.proto.PrivateKey)
	if err != nil {
		return
	}

	// Store the peer once we identify it from the handshake
	var peer *entity.Peer
	peerID := ""

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Decrypt using Noise Protocol (handshake happens on first message)
		decryptedMessage, err := session.ReadMessage(message)
		if err != nil {
			continue
		}

		// On first message, try to find peer by remote address
		if peer == nil {
			remoteAddr := r.RemoteAddr
			// Extract IP from remote address
			ip, _, err := net.SplitHostPort(remoteAddr)
			if err != nil {
				ip = remoteAddr // Fallback if parsing fails
			}

			// Find peer by IP address
			peers := l.proto.Peers.GetPeers()
			for _, p := range peers {
				if p.AddrIP == ip {
					peer = p
					peerID = p.PeerID
					// Store session in peer
					peer.Session = session
					break
				}
			}

			// If still not found, skip this message
			if peer == nil {
				continue
			}
		}

		// Add message with sender's peer ID
		peer.AddMessage(string(decryptedMessage), peerID)
	}
}

func (l *Listener) meow(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.Close()
}

func (l *Listener) Start() {
	http.HandleFunc("/chat", l.chat)
	http.HandleFunc("/meow", l.meow)

	// Retry server startup if it fails (e.g., due to network changes)
	for {
		server := &http.Server{
			Addr:    l.addr,
			Handler: nil,
		}

		err := server.ListenAndServe()
		if err != nil {
			log.Printf("listener: server error: %v, attempting to restart...", err)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
}
