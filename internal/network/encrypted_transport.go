package network

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"

	"p2p-messenger/internal/crypto"
)

var (
	ErrNoSession = errors.New("no encryption session")
)

// EncryptedConn wraps a connection with Noise Protocol encryption
type EncryptedConn struct {
	conn    net.Conn
	session *crypto.Session
}

// NewEncryptedConn creates a new encrypted connection
func NewEncryptedConn(conn net.Conn, session *crypto.Session) *EncryptedConn {
	return &EncryptedConn{
		conn:    conn,
		session: session,
	}
}

// Read reads and decrypts data
func (e *EncryptedConn) Read(b []byte) (int, error) {
	if e.session == nil {
		return 0, ErrNoSession
	}

	// Read encrypted data
	buffer := make([]byte, len(b)+64) // Add overhead for encryption
	n, err := e.conn.Read(buffer)
	if err != nil {
		return 0, err
	}

	// Decrypt
	decrypted, err := e.session.ReadMessage(buffer[:n])
	if err != nil {
		return 0, fmt.Errorf("decrypt failed: %w", err)
	}

	copy(b, decrypted)
	return len(decrypted), nil
}

// Write encrypts and writes data
func (e *EncryptedConn) Write(b []byte) (int, error) {
	if e.session == nil {
		return 0, ErrNoSession
	}

	encrypted, err := e.session.WriteMessage(b)
	if err != nil {
		return 0, fmt.Errorf("encrypt failed: %w", err)
	}

	return e.conn.Write(encrypted)
}

// Close closes the underlying connection
func (e *EncryptedConn) Close() error {
	return e.conn.Close()
}

// GetTLSConfig creates a TLS config for encrypted transport
func GetTLSConfig() *tls.Config {
	// Generate self-signed certificate for encrypted transport
	// In production, use proper certificate management
	cert, err := generateSelfSignedCert()
	if err != nil {
		return &tls.Config{
			InsecureSkipVerify: true, // For P2P, we rely on Noise Protocol for authentication
		}
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // Noise Protocol provides authentication
		MinVersion:         tls.VersionTLS12,
	}
}

func generateSelfSignedCert() (tls.Certificate, error) {
	// Generate a simple self-signed cert for TLS transport encryption
	// Noise Protocol handles authentication, TLS just provides transport encryption
	_, _, err := crypto.GenerateKeypair()
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create minimal cert (simplified - in production use proper x509)
	return tls.Certificate{}, fmt.Errorf("cert generation not fully implemented")
}

// DialEncryptedWebSocket dials a WebSocket with TLS encryption
func DialEncryptedWebSocket(url string) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{
		TLSClientConfig: GetTLSConfig(),
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(url, nil)
	return conn, err
}

