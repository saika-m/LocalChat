package ui

import (
	"github.com/rivo/tview"

	"p2p-messenger/internal/repository"
)

type Sidebar struct {
	View             *tview.List
	peerRepo         *repository.PeerRepository
	currentPeerCount int
}

func NewSidebar(peerRepo *repository.PeerRepository) *Sidebar {
	view := tview.NewList()
	view.SetTitle("peers").SetBorder(true)

	return &Sidebar{
		View:             view,
		peerRepo:         peerRepo,
		currentPeerCount: -1,
	}
}

func (s *Sidebar) Reprint() {
	peersCount := len(s.peerRepo.GetPeers())
	if s.currentPeerCount == peersCount {
		return
	}

	s.currentPeerCount = peersCount

	s.View.Clear()

	for _, peer := range s.peerRepo.GetPeers() {
		// Display peer ID (first 8 chars for brevity) instead of name for privacy
		displayID := peer.PeerID
		if len(displayID) > 8 {
			displayID = displayID[:8] + "..."
		}
		s.View.
			AddItem(displayID, peer.PeerID, 0, nil)
	}
}
