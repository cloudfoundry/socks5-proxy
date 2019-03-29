package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	proxy "github.com/cloudfoundry/socks5-proxy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
)

var _ = Describe("StartTestSSHServer", func() {
	var (
		hostPort     string
		clientConfig *ssh.ClientConfig
		interval     time.Duration
	)

	BeforeEach(func() {
		interval = 10 * time.Millisecond
		httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))
		hostPort = strings.Split(httpServer.URL, "http://")[1]

		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		Expect(err).NotTo(HaveOccurred())

		clientConfig = &ssh.ClientConfig{
			User: "jumpbox",
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.FixedHostKey(signer.PublicKey()),
		}
	})

	It("accepts multiple requests", func() {
		url := proxy.StartTestSSHServer(hostPort, privateKey, "", interval)

		conn1, err := ssh.Dial("tcp", url, clientConfig)
		Expect(err).NotTo(HaveOccurred())
		conn1.Close()

		conn2, err := ssh.Dial("tcp", url, clientConfig)
		Expect(err).NotTo(HaveOccurred())
		conn2.Close()
	})
})
