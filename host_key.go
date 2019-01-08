package proxy

import (
	"net"

	"golang.org/x/crypto/ssh"
)

type HostKey struct{}

func NewHostKey() HostKey {
	return HostKey{}
}

func (h HostKey) Get(username, privateKey, serverURL string) (ssh.PublicKey, error) {
	publicKeyChannel := make(chan ssh.PublicKey, 1)

	if username == "" {
		username = "jumpbox"
	}

	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	clientConfig := NewSSHClientConfig(username, keyScanCallback(publicKeyChannel), ssh.PublicKeys(signer))

	conn, err := ssh.Dial("tcp", serverURL, clientConfig)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return <-publicKeyChannel, nil
}

func keyScanCallback(publicKeyChannel chan ssh.PublicKey) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		publicKeyChannel <- key
		return nil
	}
}
