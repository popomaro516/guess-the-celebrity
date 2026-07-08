package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

type Generator struct{}

func New() Generator {
	return Generator{}
}

func (Generator) NewID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
