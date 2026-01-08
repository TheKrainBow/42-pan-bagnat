package core

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

var ErrInvalidSSHKey = errors.New("invalid ssh private key")

func GenerateSSHKeys() (PublicKey string, PrivateKey string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ssh key: %w", err)
	}

	privBuf := new(bytes.Buffer)
	if err := pem.Encode(privBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}
	pub, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ssh public key: %w", err)
	}
	pubStr := string(ssh.MarshalAuthorizedKey(pub))
	return strings.TrimSpace(pubStr), strings.TrimSpace(privBuf.String()), nil
}

// DeriveSSHPublicKey takes an OpenSSH-compatible private key and derives the
// matching authorized_keys formatted public key.
func DeriveSSHPublicKey(privateKey string) (string, error) {
	trimmed := strings.TrimSpace(privateKey)
	if trimmed == "" {
		return "", ErrInvalidSSHKey
	}
	signer, err := ssh.ParsePrivateKey([]byte(trimmed))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidSSHKey, err)
	}
	pub := signer.PublicKey()
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub))), nil
}
