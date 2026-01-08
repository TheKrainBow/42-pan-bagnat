package sshkeys

// SSHKeyCreateInput describes the payload to create a new SSH key.
type SSHKeyCreateInput struct {
	Name          string `json:"name" example:"shared-deploy-key"`
	SSHPrivateKey string `json:"ssh_private_key,omitempty" example:"-----BEGIN OPENSSH PRIVATE KEY-----"`
}

// SSHKeyRegenerateInput allows providing a replacement private key when regenerating.
type SSHKeyRegenerateInput struct {
	SSHPrivateKey string `json:"ssh_private_key,omitempty" example:"-----BEGIN OPENSSH PRIVATE KEY-----"`
}
