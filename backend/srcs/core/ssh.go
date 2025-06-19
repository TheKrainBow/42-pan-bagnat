package core

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

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
