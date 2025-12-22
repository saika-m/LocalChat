package network

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"p2p-messenger/internal/proto"
)

func TestDiscoverer(t *testing.T) {
	// Create two protos
	proto1, err := proto.NewProto("25043")
	assert.NoError(t, err)
	proto2, err := proto.NewProto("25044")
	assert.NoError(t, err)

	// Create two discoverers
	multicastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", MulticastIP, "25043"))
	assert.NoError(t, err)
	discoverer1 := NewDiscoverer(multicastAddr, 100*time.Millisecond, proto1)
	discoverer2 := NewDiscoverer(multicastAddr, 100*time.Millisecond, proto2)

	// Start discoverers
	go discoverer1.Start()
	go discoverer2.Start()

	// Wait for discovery
	time.Sleep(500 * time.Millisecond)

	// Check if peers are discovered
	peers1 := proto1.Peers.GetPeers()
	peers2 := proto2.Peers.GetPeers()

	assert.Equal(t, 1, len(peers1))
	assert.Equal(t, 1, len(peers2))

	assert.Equal(t, proto2.PublicKeyStr, string(peers1[0].PublicKey))
	assert.Equal(t, proto1.PublicKeyStr, string(peers2[0].PublicKey))
}
