package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"

	// Import os/exec for font-size AppleScript
	"os"
	"os/exec"
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

	// Launch network manager and set terminal font size via AppleScript
	runNetworkManager(p)
	fontCmd := exec.Command("osascript", "-e", `tell application "Terminal" to set font size of window 1 to 14`)
	if err := fontCmd.Run(); err != nil {
		log.Printf("Failed to set font size: %v", err)
	}

	// Resize terminal window to 39 rows × 139 columns
	fmt.Print("\033[8;39;139t")

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
