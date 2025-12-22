package dht

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/multiformats/go-multiaddr"
)

func TestManager(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Peer found callback
	var foundPeer1, foundPeer2 bool
	peerFoundCb1 := func(peerID string, addrs []multiaddr.Multiaddr) {
		foundPeer1 = true
	}
	peerFoundCb2 := func(peerID string, addrs []multiaddr.Multiaddr) {
		foundPeer2 = true
	}

	// Create two managers
	manager1, err := NewManager(25047, peerFoundCb1)
	assert.NoError(t, err)
	manager2, err := NewManager(25048, peerFoundCb2)
	assert.NoError(t, err)

	// Start managers
	manager1.Start()
	manager2.Start()

	// Wait for discovery
	select {
	case <-time.After(25 * time.Second):
		assert.True(t, foundPeer1)
		assert.True(t, foundPeer2)
	case <-ctx.Done():
		t.Fatal("Test timed out")
	}
}
