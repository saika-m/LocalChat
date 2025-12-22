package repository

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"p2p-messenger/internal/entity"
)

const (
	peerValidationTimeOut = 1 * time.Second
)

type PeerRepository struct {
	rwMutex *sync.RWMutex
	peers   map[string]*entity.Peer
}

func NewPeerRepository() *PeerRepository {
	peerRepository := &PeerRepository{
		rwMutex: &sync.RWMutex{},
		peers:   make(map[string]*entity.Peer),
	}

	peerRepository.peersValidator()

	return peerRepository
}

func (p *PeerRepository) Add(peer *entity.Peer) {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	existing, found := p.peers[peer.PeerID]
	if !found {
		p.peers[peer.PeerID] = peer
	} else {
		// Merge connection types - add new connection types if they don't exist
		for _, ct := range peer.ConnectionTypes {
			existing.AddConnectionType(ct)
		}
		// Update other fields if they're missing
		if existing.AddrIP == "" && peer.AddrIP != "" {
			existing.AddrIP = peer.AddrIP
		}
		if existing.Port == "" && peer.Port != "" {
			existing.Port = peer.Port
		}
		if existing.BLEAddr == "" && peer.BLEAddr != "" {
			existing.BLEAddr = peer.BLEAddr
		}
		if len(existing.PublicKey) == 0 && len(peer.PublicKey) > 0 {
			existing.PublicKey = peer.PublicKey
		}
	}
}

func (p *PeerRepository) Delete(peerID string) {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	delete(p.peers, peerID)
}

func (p *PeerRepository) Get(peerID string) (*entity.Peer, bool) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	peer, found := p.peers[peerID]
	return peer, found
}

func (p *PeerRepository) GetPeers() []*entity.Peer {
	peersSlice := make([]*entity.Peer, 0, len(p.peers))

	for _, peer := range p.peers {
		peersSlice = append(peersSlice, peer)
	}

	sort.Slice(peersSlice, func(i, j int) bool {
		return peersSlice[i].PeerID < peersSlice[j].PeerID
	})

	return peersSlice
}

func (p *PeerRepository) peersValidator() {
	ticker := time.NewTicker(peerValidationTimeOut)

	go func() {
		for {
			<-ticker.C
			// Make a copy of peers to avoid holding lock during network operations
			p.rwMutex.RLock()
			peersCopy := make([]*entity.Peer, 0, len(p.peers))
			for _, peer := range p.peers {
				peersCopy = append(peersCopy, peer)
			}
			p.rwMutex.RUnlock()

			for _, peer := range peersCopy {
				if peer.PrimaryConnectionType == entity.ConnectionBLE {
					continue
				}
				if peer.AddrIP == "" || peer.Port == "" {
					continue
				}

				u := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%s", peer.AddrIP, peer.Port), Path: "/meow"}

				// Use a short timeout to avoid hanging on network issues
				dialer := &websocket.Dialer{
					HandshakeTimeout: 2 * time.Second,
				}

				c, _, err := dialer.Dial(u.String(), nil)
				if c == nil || err != nil {
					// Only delete if it's a permanent error, not temporary network issues
					if err != nil {
						// Check if it's a network error that might be temporary
						if netErr, ok := err.(net.Error); ok {
							if netErr.Timeout() || netErr.Temporary() {
								// Temporary network issue, don't delete peer
								continue
							}
						}
					}
					// Only delete if connection truly failed (not timeout/temporary)
					p.Delete(peer.PeerID)
					continue
				}
				c.Close()
			}
		}
	}()
}
