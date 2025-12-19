package network

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"p2p-messenger/internal/crypto"
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

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Decrypt using Noise Protocol
		decryptedMessage, err := session.ReadMessage(message)
		if err != nil {
			continue
		}

		// Find peer by public key (extracted from handshake)
		// For now, use a simplified approach - in production, extract peer ID from handshake
		peerID := crypto.PeerID(l.proto.PublicKey)
		peer, found := l.proto.Peers.Get(peerID)
		if !found {
			continue
		}

		peer.AddMessage(string(decryptedMessage), peer.PeerID)
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
	log.Fatal(http.ListenAndServe(l.addr, nil))
}
