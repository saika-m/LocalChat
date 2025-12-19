package repository

import (
	"fmt"
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

	_, found := p.peers[peer.PeerID]
	if !found {
		p.peers[peer.PeerID] = peer
	}
}

func (p *PeerRepository) Delete(peerID string) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

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
			for _, peer := range p.peers {
				if peer.AddrIP == "" {
					continue
				}
				u := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%s", peer.AddrIP, peer.Port), Path: "/meow"}

				c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
				if c == nil {
					p.Delete(peer.PeerID)
					continue
				}
				c.Close()
			}
		}
	}()
}
