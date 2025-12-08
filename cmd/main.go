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
	name := flag.String("name", "", "peer's name")
	flag.Parse()

	// If name not provided via flag, prompt for it
	if *name == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your name: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading name: %v", err)
		}
		*name = strings.TrimSpace(input)
		if *name == "" {
			log.Fatal("Name cannot be empty")
		}
	}

	p := proto.NewProto(*name, Port)

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
