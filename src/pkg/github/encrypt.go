package github

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/google/go-github/v60/github"
	"golang.org/x/crypto/nacl/box"
)

// encryptSecretWithPublicKey encrypts a secret using GitHub's public key
func encryptSecretWithPublicKey(secret []byte, publicKey *github.PublicKey) (string, error) {
	decodedPubKey, err := base64.StdEncoding.DecodeString(publicKey.GetKey())
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}
	var peersPubKey [32]byte
	copy(peersPubKey[:], decodedPubKey[0:32])

	var rand io.Reader
	eBody, err := box.SealAnonymous(nil, secret[:], &peersPubKey, rand)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt body: %w", err)

	}

	//
	return base64.StdEncoding.EncodeToString(eBody), nil
}
