package entity

import (
	b "bytes"
	"errors"
	"strings"
)

const (
	nullByte = "\x00"
)

var (
	ErrBadMulticastMessage = errors.New("ErrorBadMulticastMessage")
)

type MulticastMessage struct {
	MulticastString string
	PubKeyStr       string
	Port            string
}

func UDPMulticastMessageToPeer(bytes []byte) (*MulticastMessage, error) {
	bytes = b.Trim(bytes, nullByte)
	array := strings.Split(string(bytes), ":")

	if len(array) != 3 {
		return nil, ErrBadMulticastMessage
	}

	return &MulticastMessage{
		MulticastString: array[0],
		PubKeyStr:       array[1],
		Port:            array[2],
	}, nil
}
