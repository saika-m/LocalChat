package network

import (
	"fmt"
	"log"
	"net"
	"time"

	"p2p-messenger/internal/entity"
	"p2p-messenger/internal/proto"
	"p2p-messenger/pkg/udp"
)

const (
	udpConnectionBufferSize = 1024
	multicastString         = "me0w"
)

type Discoverer struct {
	Addr               *net.UDPAddr
	MulticastFrequency time.Duration
	Proto              *proto.Proto
}

func NewDiscoverer(addr *net.UDPAddr, multicastFrequency time.Duration, proto *proto.Proto) *Discoverer {
	return &Discoverer{
		Addr:               addr,
		MulticastFrequency: multicastFrequency,
		Proto:              proto,
	}
}

func (d *Discoverer) Start() {
	go d.startMulticasting()
	go d.listenMulticasting()
}

func (d *Discoverer) startMulticasting() {
	conn, err := net.DialUDP("udp", nil, d.Addr)
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(d.MulticastFrequency)
	for {
		<-ticker.C
		// Minimal metadata: only public key and port (no name for privacy)
		_, err := conn.Write([]byte(fmt.Sprintf("%s:%s:%s",
			multicastString,
			d.Proto.PublicKeyStr,
			d.Proto.Port)))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (d *Discoverer) listenMulticasting() {
	conn, err := net.ListenMulticastUDP("udp", nil, d.Addr)
	if err != nil {
		log.Fatal(err)
	}

	err = conn.SetReadBuffer(udpConnectionBufferSize)
	if err != nil {
		log.Fatal(err)
	}

	for {
		rawBytes, addr, err := udp.ReadFromUDPConnection(conn, udpConnectionBufferSize)
		if err != nil {
			log.Fatal(err)
		}

		message, err := entity.UDPMulticastMessageToPeer(rawBytes)
		if err != nil {
			log.Fatal(err)
		}

		// Convert public key string to bytes
		pubKeyBytes := []byte(message.PubKeyStr)
		if len(pubKeyBytes) != 32 {
			continue
		}

		peerID := entity.PeerIDFromPublicKey(pubKeyBytes)
		peer := &entity.Peer{
			PeerID:    peerID,
			PublicKey: pubKeyBytes,
			Port:      message.Port,
			Messages:  make([]*entity.Message, 0),
			AddrIP:    addr.IP.String(),
		}

		if message.PubKeyStr != d.Proto.PublicKeyStr {
			d.Proto.Peers.Add(peer)
		}
	}
}
