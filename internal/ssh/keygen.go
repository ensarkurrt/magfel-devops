package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func GenerateKeyPair(keyPath string) error {
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	privBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return fmt.Errorf("marshaling private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(privBytes)
	if err := os.WriteFile(keyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("converting public key: %w", err)
	}

	pubBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	pubPath := keyPath + ".pub"
	if err := os.WriteFile(pubPath, pubBytes, 0644); err != nil {
		return fmt.Errorf("writing public key: %w", err)
	}

	return nil
}

func KeyExists(keyPath string) bool {
	_, err := os.Stat(keyPath)
	return err == nil
}

func ReadPublicKey(keyPath string) (string, error) {
	pubPath := keyPath + ".pub"
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return "", fmt.Errorf("reading public key: %w", err)
	}
	return string(data), nil
}
