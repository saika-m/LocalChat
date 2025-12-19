package bluetooth

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/darwin"

	"p2p-messenger/internal/crypto"
	"p2p-messenger/internal/entity"
)

const (
	bleServiceUUIDStr     = "6e400001-b5a3-f393-e0a9-e50e24dcca9e"
	bleMetaCharacteristic = "6e400002-b5a3-f393-e0a9-e50e24dcca9e"

	connectionTimeout = 5 * time.Second
)

// Manager hosts a BLE GATT service that advertises peer metadata and scans for
// nearby peers to support offline/local discovery.
type Manager struct {
	serviceUUID ble.UUID
	metaUUID    ble.UUID
	proto       *entityProto
	stop        context.CancelFunc
}

// entityProto is a minimal subset of proto.Proto to avoid import cycles.
type entityProto struct {
	PublicKeyStr string
	Port         string
	Peers        peerRepository
}

// peerRepository matches the public methods we need from repository.PeerRepository.
type peerRepository interface {
	Add(peer *entity.Peer)
	Get(peerID string) (*entity.Peer, bool)
}

func NewManager(publicKeyStr string, port string, peers peerRepository) *Manager {
	return &Manager{
		serviceUUID: ble.MustParse(bleServiceUUIDStr),
		metaUUID:    ble.MustParse(bleMetaCharacteristic),
		proto: &entityProto{
			PublicKeyStr: publicKeyStr,
			Port:         port,
			Peers:        peers,
		},
	}
}

// Start begins advertising and scanning; errors are logged but not fatal.
func (m *Manager) Start() {
	dev, err := darwin.NewDevice()
	if err != nil {
		log.Printf("bluetooth: skipping BLE, unable to init device: %v", err)
		return
	}
	ble.SetDefaultDevice(dev)

	if err := m.addService(); err != nil {
		log.Printf("bluetooth: unable to register service: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.stop = cancel

	go m.advertise(ctx)
	go m.scan(ctx)
}

func (m *Manager) Stop() {
	if m.stop != nil {
		m.stop()
	}
}

func (m *Manager) advertise(ctx context.Context) {
	// Advertise only service UUID (no name for privacy); blocks until ctx is cancelled.
	if err := ble.AdvertiseNameAndServices(ctx, "", m.serviceUUID); err != nil {
		log.Printf("bluetooth: advertise stopped: %v", err)
	}
}

func (m *Manager) scan(ctx context.Context) {
	filter := func(a ble.Advertisement) bool {
		return a.Connectable() && hasService(a, m.serviceUUID)
	}

	for {
		err := ble.Scan(ctx, false, m.handleAdvertisement, filter)
		if err != nil && ctx.Err() == nil {
			log.Printf("bluetooth: scan error: %v", err)
			time.Sleep(time.Second)
		}

		if ctx.Err() != nil {
			return
		}
	}
}

func (m *Manager) handleAdvertisement(a ble.Advertisement) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	client, err := ble.Dial(ctx, a.Addr())
	if err != nil {
		return
	}
	defer client.CancelConnection()

	characteristic, err := m.findMetaCharacteristic(ctx, client)
	if err != nil || characteristic == nil {
		return
	}

	data, err := client.ReadCharacteristic(characteristic)
	if err != nil || len(data) == 0 {
		return
	}

	meta, err := parseMetadata(string(data))
	if err != nil {
		return
	}

	// Ignore our own advertisements.
	if meta.PubKeyStr == m.proto.PublicKeyStr {
		return
	}

	// Convert public key string to bytes
	pubKeyBytes := []byte(meta.PubKeyStr)
	if len(pubKeyBytes) != 32 {
		// Try to parse as hex if needed
		return
	}

	peerID := crypto.PeerID(pubKeyBytes)
	peer := &entity.Peer{
		PeerID:    peerID,
		PublicKey: pubKeyBytes,
		Port:      meta.Port,
		Messages:  make([]*entity.Message, 0),
		BLEAddr:   a.Addr().String(),
	}

	m.proto.Peers.Add(peer)
}

func (m *Manager) findMetaCharacteristic(ctx context.Context, client ble.Client) (*ble.Characteristic, error) {
	profile, err := client.DiscoverProfile(true)
	if err != nil {
		return nil, err
	}

	for _, s := range profile.Services {
		if !s.UUID.Equal(m.serviceUUID) {
			continue
		}
		for _, c := range s.Characteristics {
			if c.UUID.Equal(m.metaUUID) {
				return c, nil
			}
		}
	}

	return nil, fmt.Errorf("metadata characteristic not found")
}

func (m *Manager) addService() error {
	service := ble.NewService(m.serviceUUID)

	metaChar := ble.NewCharacteristic(m.metaUUID)
	metaChar.HandleRead(ble.ReadHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) {
		_, _ = rsp.Write([]byte(m.metadataPayload()))
	}))

	service.AddCharacteristic(metaChar)
	return ble.AddService(service)
}

func (m *Manager) metadataPayload() string {
	// Minimal metadata: only public key and port (no name, no IP for privacy)
	// Public key is necessary for Noise Protocol handshake
	return strings.Join([]string{
		m.proto.PublicKeyStr,
		m.proto.Port,
	}, "|")
}

type metadata struct {
	PubKeyStr string
	Port      string
}

func parseMetadata(payload string) (*metadata, error) {
	parts := strings.Split(payload, "|")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid metadata payload")
	}

	return &metadata{
		PubKeyStr: parts[0],
		Port:      parts[1],
	}, nil
}

func hasService(a ble.Advertisement, uuid ble.UUID) bool {
	for _, srv := range a.Services() {
		if srv.Equal(uuid) {
			return true
		}
	}
	return false
}
