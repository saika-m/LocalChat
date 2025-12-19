package proto

import (
	"p2p-messenger/internal/crypto"
	"p2p-messenger/internal/repository"
)

type Proto struct {
	// PublicKeyStr is the Noise Protocol public key as string (for metadata)
	PublicKeyStr string
	// PublicKey is the Noise Protocol public key bytes
	PublicKey []byte
	// PrivateKey is the Noise Protocol private key (for responder sessions)
	PrivateKey crypto.NoiseKeypair
	Peers      *repository.PeerRepository
	Port       string
}

func NewProto(port string) (*Proto, error) {
	keypair, pubKey, err := crypto.GenerateKeypair()
	if err != nil {
		return nil, err
	}

	return &Proto{
		PublicKeyStr: string(pubKey),
		PublicKey:    pubKey,
		PrivateKey:   keypair,
		Peers:        repository.NewPeerRepository(),
		Port:         port,
	}, nil
}
