# _simple p2p messenger_

## Running from Source

Run
```sh
go run cmd/main.go -name saika-m
```

Or without the flag to be prompted for your name:
```sh
go run cmd/main.go
```

## Packaging for macOS

To build a macOS app bundle:

```sh
./build-mac.sh
```

This will create `build/P2P Chat.app`. You can:
- Run it directly: `open "build/P2P Chat.app"`
- Or drag it to your Applications folder

When you run the app, it will:
1. Open Terminal
2. Prompt you to enter your name
3. Start the P2P chat application

## About

My messenger discovers users using UDP, connects to them using websocket to encrypt/decrypt messages uses diffie-hellman algorithm. For ui i used tview.
