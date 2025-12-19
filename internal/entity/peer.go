package entity

import (
	"errors"
	"fmt"
	"time"

	"p2p-messenger/internal/crypto"
)

var (
	ErrPeerIsDeleted = errors.New("peer disconnected")
)

// Peer represents a discovered peer with minimal metadata for privacy
type Peer struct {
	// PeerID is a privacy-preserving identifier (hash of public key)
	PeerID string
	// PublicKey is the Noise Protocol public key (32 bytes)
	PublicKey []byte
	// Messages stores encrypted messages
	Messages []*Message
	// AddrIP is optional IP address (only if available)
	AddrIP string
	// Port is the listening port
	Port string
	// BLEAddr is optional BLE address
	BLEAddr string
	// Session holds the Noise Protocol session for E2EE
	Session *crypto.Session
}

func (p *Peer) AddMessage(text, author string) {
	p.Messages = append(p.Messages, &Message{
		Time:   time.Now(),
		Text:   text,
		Author: author,
	})
}

// SendMessage sends an encrypted message using Noise Protocol
func (p *Peer) SendMessage(message string) error {
	if p.Session == nil {
		return errors.New("no session established")
	}

	encrypted, err := p.Session.WriteMessage([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	// TODO: Send over encrypted transport (will be implemented in listener/transport layer)
	_ = encrypted
	return nil
}

// PeerIDFromPublicKey generates a peer ID from a public key
func PeerIDFromPublicKey(pubKey []byte) string {
	return crypto.PeerID(pubKey)
}
