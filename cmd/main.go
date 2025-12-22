package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"p2p-messenger/internal/network"
	"p2p-messenger/internal/proto"
	"p2p-messenger/internal/ui"
)

const (
	Port = "25042"
)

func main() {
	flag.Parse()

	f, err := os.OpenFile("p2p-chat.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	fmt.Print("Please type in your name: ")
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read username: %v", err)
	}
	username = strings.TrimSpace(username)

	p, err := proto.NewProto(Port)
	if err != nil {
		log.Fatalf("Failed to create proto: %v", err)
	}
	p.SetUsername(username)

	runNetworkManager(p)

	if err := runUI(p); err != nil {
		log.Fatal(err)
	}
}

func runNetworkManager(p *proto.Proto) *network.Manager {
	networkManager := network.NewManager(p)
	p.NetworkManager = networkManager
	networkManager.Start()
	return networkManager
}

func runUI(p *proto.Proto) error {
	return ui.NewApp(p).Run()
}
