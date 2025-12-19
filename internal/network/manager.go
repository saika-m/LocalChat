package network

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"p2p-messenger/internal/bluetooth"
	"p2p-messenger/internal/dht"
	"p2p-messenger/internal/proto"
)

const (
	MulticastIP        = "224.0.0.1"
	ListenerIP         = "0.0.0.0"
	MulticastFrequency = 1 * time.Second
)

type Manager struct {
	Proto      *proto.Proto
	Listener   *Listener
	Discoverer *Discoverer
	BLE        *bluetooth.Manager
	DHT        *dht.Manager
}

func NewManager(proto *proto.Proto) *Manager {
	multicastAddr, err := net.ResolveUDPAddr(
		"udp",
		fmt.Sprintf("%s:%s", MulticastIP, proto.Port))
	if err != nil {
		log.Fatal(err)
	}

	listenerAddr := fmt.Sprintf("%s:%s", ListenerIP, proto.Port)

	portInt, err := strconv.Atoi(proto.Port)
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}

	dhtManager, err := dht.NewManager(portInt + 1) // Use different port for DHT
	if err != nil {
		log.Printf("Warning: DHT initialization failed: %v", err)
	}

	return &Manager{
		Proto:      proto,
		Listener:   NewListener(listenerAddr, proto),
		Discoverer: NewDiscoverer(multicastAddr, MulticastFrequency, proto),
		BLE:        bluetooth.NewManager(proto.PublicKeyStr, proto.Port, proto.Peers),
		DHT:        dhtManager,
	}
}

func (m *Manager) Start() {
	go m.Listener.Start()
	go m.Discoverer.Start()
	if m.BLE != nil {
		go m.BLE.Start()
	}
	if m.DHT != nil {
		m.DHT.Start()
	}
}
