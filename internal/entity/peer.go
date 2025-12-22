package entity

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"p2p-messenger/internal/crypto"
)

var (
	ErrPeerIsDeleted = errors.New("peer disconnected")
)

const (
	bleServiceUUIDStr     = "6e400001-b5a3-f393-e0a9-e50e24dcca9e"
	bleMetaCharacteristic = "6e400002-b5a3-f393-e0a9-e50e24dcca9e"
	connectionTimeout     = 5 * time.Second
)

// ConnectionType represents how a peer is connected
type ConnectionType int

const (
	ConnectionBLE ConnectionType = iota
	ConnectionNAT
	ConnectionInternet
)

func (ct ConnectionType) String() string {
	switch ct {
	case ConnectionBLE:
		return "BLE"
	case ConnectionNAT:
		return "NAT"
	case ConnectionInternet:
		return "Internet"
	default:
		return "Unknown"
	}
}

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
	// ConnectionTypes tracks all available connection types for this peer (prioritized)
	ConnectionTypes []ConnectionType
	// PrimaryConnectionType is the preferred connection type (highest priority)
	PrimaryConnectionType ConnectionType
	// Session holds the Noise Protocol session for E2EE
	Session *crypto.Session
	// conn holds the WebSocket connection
	conn     *websocket.Conn
	connLock sync.Mutex
}

func (p *Peer) AddMessage(text, author string) {
	p.Messages = append(p.Messages, &Message{
		Time:   time.Now(),
		Text:   text,
		Author: author,
	})
}

// EstablishConnection establishes a WebSocket connection and Noise Protocol session
func (p *Peer) EstablishConnection() error {
	if p.AddrIP == "" || p.Port == "" {
		return errors.New("peer address not available")
	}

	p.connLock.Lock()
	defer p.connLock.Unlock()

	// If already connected, return
	if p.conn != nil {
		return nil
	}

	// Establish WebSocket connection
	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%s", p.AddrIP, p.Port), Path: "/chat"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	// Create Noise Protocol session as initiator
	session, _, err := crypto.NewInitiatorSession()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create initiator session: %w", err)
	}

	// Perform handshake by sending an empty message (handshake will happen on first WriteMessage)
	p.Session = session
	p.conn = conn

	return nil
}

// SendMessage sends an encrypted message using Noise Protocol
func (p *Peer) SendMessage(message string) error {
	p.connLock.Lock()
	defer p.connLock.Unlock()

	// Establish connection if needed
	if p.conn == nil || p.Session == nil {
		if err := p.EstablishConnection(); err != nil {
			return fmt.Errorf("failed to establish connection: %w", err)
		}
	}

	// Encrypt message using Noise Protocol
	encrypted, err := p.Session.WriteMessage([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Send over WebSocket
	if err := p.conn.WriteMessage(websocket.BinaryMessage, encrypted); err != nil {
		// Connection might be broken, close it
		p.conn.Close()
		p.conn = nil
		p.Session = nil
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Close closes the WebSocket connection
func (p *Peer) Close() {
	p.connLock.Lock()
	defer p.connLock.Unlock()
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
		p.Session = nil
	}
}

// AddConnectionType adds a connection type and updates primary if needed
func (p *Peer) AddConnectionType(ct ConnectionType) {
	// Initialize slice if needed
	if p.ConnectionTypes == nil {
		p.ConnectionTypes = make([]ConnectionType, 0)
	}
	
	// Check if already exists
	for _, existing := range p.ConnectionTypes {
		if existing == ct {
			return
		}
	}
	
	// Add connection type
	p.ConnectionTypes = append(p.ConnectionTypes, ct)
	
	// Update primary connection type based on priority: BLE > NAT > Internet
	if len(p.ConnectionTypes) == 1 {
		p.PrimaryConnectionType = ct
	} else {
		// Prioritize: BLE (0) < NAT (1) < Internet (2) - lower number = higher priority
		if ct < p.PrimaryConnectionType {
			p.PrimaryConnectionType = ct
		}
	}
}

// PeerIDFromPublicKey generates a peer ID from a public key
func PeerIDFromPublicKey(pubKey []byte) string {
	return crypto.PeerID(pubKey)
}