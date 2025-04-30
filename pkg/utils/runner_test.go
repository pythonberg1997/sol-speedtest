package utils

import (
	"testing"
	"github.com/gagliardetto/solana-go"
	"fmt"
	"crypto/sha256"
)

func TestNonce(t *testing.T) {
	pre := "BaPG4TTXoWi8UN3qUcDzGQM55Az5rkzFxtsST3EdThsm"
	preHash, err := solana.PublicKeyFromBase58(pre)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}

	post := sha256.Sum256(preHash.Bytes())
	postHash := solana.PublicKeyFromBytes(post[:])
	fmt.Printf("post hash: %s\n", postHash.String())
}
