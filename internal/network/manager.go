package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/multiformats/go-multiaddr"

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

	// Cached availability status (updated periodically)
	bleAvailable      bool
	natAvailable      bool
	internetAvailable bool
	lastCheck         time.Time
	checkMutex        sync.Mutex
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

	// DHT callback to add discovered peers
	dhtPeerFoundCb := func(peerID string, addrs []multiaddr.Multiaddr) {
		// Extract IP and port from multiaddr
		for _, addr := range addrs {
			// Try to extract /ip4 or /ip6 address and /tcp port
			ip := ""
			port := ""
			addrStr := addr.String()
			// Parse multiaddr format: /ip4/192.168.1.1/tcp/1234
			// This is simplified - in production, use proper multiaddr parsing
			if len(addrStr) > 0 {
				// For now, we'll use mDNS discovered peers which have IP addresses
				// DHT peers need libp2p connection handling which is more complex
				// So we'll focus on mDNS for now
			}
			_ = ip
			_ = port
		}
	}

	dhtManager, err := dht.NewManager(portInt+1, dhtPeerFoundCb) // Use different port for DHT
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
		// Give BLE manager a moment to initialize before checking
		// This prevents race condition when Bluetooth is already on at startup
		time.Sleep(100 * time.Millisecond)
	}
	if m.DHT != nil {
		m.DHT.Start()
	}

	// Do initial availability check after BLE has had time to initialize
	m.updateAvailability()

	// Start periodic availability checking
	go m.checkAvailabilityPeriodically()
}

// checkAvailabilityPeriodically checks availability of each mode every second
func (m *Manager) checkAvailabilityPeriodically() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		m.updateAvailability()
	}
}

// updateAvailability checks and updates the availability status of all modes
func (m *Manager) updateAvailability() {
	// Get current values first (to preserve on error for other checks)
	m.checkMutex.Lock()
	currentNat := m.natAvailable
	currentInternet := m.internetAvailable
	m.checkMutex.Unlock()

	// Check BLE availability FIRST and update immediately
	// This is completely independent and must always work, even when WiFi is off
	var bleAvail bool
	if m.BLE != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("network: BLE availability check panic: %v", r)
					// On panic, read current value
					m.checkMutex.Lock()
					bleAvail = m.bleAvailable
					m.checkMutex.Unlock()
				}
			}()
			// Do BLE check in complete isolation - this MUST always work
			bleAvail = m.BLE.IsAvailable()
		}()
	} else {
		bleAvail = false
	}

	// Update BLE immediately (don't wait for other checks)
	m.checkMutex.Lock()
	m.bleAvailable = bleAvail
	m.checkMutex.Unlock()

	// Now check other modes (these can fail without affecting BLE)
	var natAvail, internetAvail bool

	// Check NAT/multicast availability (completely independent)
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("network: NAT availability check panic: %v", r)
				natAvail = currentNat
			}
		}()
		natAvail = m.checkNATAvailable()
	}()

	// Check Internet availability (completely independent, runs last)
	// This is the slowest check and should not affect others
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("network: Internet availability check panic: %v", r)
				internetAvail = currentInternet
			}
		}()
		internetAvail = m.checkInternetAvailable()
	}()

	// Update NAT and Internet atomically
	m.checkMutex.Lock()
	m.natAvailable = natAvail
	m.internetAvailable = internetAvail
	m.lastCheck = time.Now()
	m.checkMutex.Unlock()
}

// checkNATAvailable checks if NAT/multicast is possible on current network
// Requires: active network connection AND multicast support
func (m *Manager) checkNATAvailable() bool {
	// First, check if we have an active network interface with an IP address
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	hasActiveInterface := false
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// Check if interface has a non-loopback IP address
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				hasActiveInterface = true
				break
			}
		}

		if hasActiveInterface {
			break
		}
	}

	if !hasActiveInterface {
		return false
	}

	// Now check if multicast actually works
	// Try to create a test multicast connection
	// If multicast is blocked (like on school WiFi), this will fail
	testAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", MulticastIP, m.Proto.Port))
	if err != nil {
		return false
	}

	// Try to listen on multicast address
	conn, err := net.ListenMulticastUDP("udp", nil, testAddr)
	if err != nil {
		return false
	}
	conn.Close()

	// Try to send to multicast address
	sendConn, err := net.DialUDP("udp", nil, testAddr)
	if err != nil {
		return false
	}
	sendConn.Close()

	return true
}

// checkInternetAvailable pings 8.8.8.8 to check internet connectivity
func (m *Manager) checkInternetAvailable() bool {
	// Use ping with timeout (1 second)
	// macOS: -c 1 (count), -W 1000 (wait timeout in milliseconds)
	// Linux: -c 1 (count), -W 1 (wait timeout in seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1000", "8.8.8.8")
	err := cmd.Run()
	if err == nil {
		return true
	}

	// If context timeout, definitely not available
	if ctx.Err() == context.DeadlineExceeded {
		return false
	}

	// Try Linux format as fallback
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()
	cmd = exec.CommandContext(ctx2, "ping", "-c", "1", "-W", "1", "8.8.8.8")
	err = cmd.Run()
	return err == nil && ctx2.Err() == nil
}

// GetAvailableModes returns which connection modes are currently available
func (m *Manager) GetAvailableModes() (bleAvailable, natAvailable, internetAvailable bool) {
	// Check if we need to update (without holding lock during check)
	needsUpdate := false
	m.checkMutex.Lock()
	if time.Since(m.lastCheck) > 500*time.Millisecond {
		needsUpdate = true
	}
	m.checkMutex.Unlock()

	// Do update outside of lock to avoid blocking
	if needsUpdate {
		m.updateAvailability()
	}

	// Return current values
	m.checkMutex.Lock()
	defer m.checkMutex.Unlock()
	return m.bleAvailable, m.natAvailable, m.internetAvailable
}
