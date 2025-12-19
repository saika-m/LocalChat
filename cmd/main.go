package main

import (
	"flag"
	"log"

	"p2p-messenger/internal/network"
	"p2p-messenger/internal/proto"
	"p2p-messenger/internal/ui"
)

const (
	Port = "25042"
)

func main() {
	flag.Parse()

	// Create proto without name (privacy-focused: use peer ID instead)
	p, err := proto.NewProto(Port)
	if err != nil {
		log.Fatalf("Failed to create proto: %v", err)
	}

	runNetworkManager(p)

	if err := runUI(p); err != nil {
		log.Fatal(err)
	}
}

func runNetworkManager(p *proto.Proto) {
	networkManager := network.NewManager(p)
	networkManager.Start()
}

func runUI(p *proto.Proto) error {
	return ui.NewApp(p).Run()
}
