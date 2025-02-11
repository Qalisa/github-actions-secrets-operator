package github

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// encryptSecretWithPublicKey encrypts a secret using GitHub's public key
func encryptSecretWithPublicKey(secret []byte, publicKey string) (string, error) {
	// Decode the public key
	decodedPublicKey, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// Convert to the correct format
	var ghPubKey [32]byte
	copy(ghPubKey[:], decodedPublicKey)

	// Generate a random nonce
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Generate ephemeral key pair
	_, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Encrypt the secret using libsodium box
	encryptedBytes := box.Seal(nonce[:], secret, &nonce, &ghPubKey, priv)

	// Encode the result in base64
	encoded := base64.StdEncoding.EncodeToString(encryptedBytes)
	return encoded, nil
}
