package proxy_test

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
	goproxy "golang.org/x/net/proxy"

	proxy "github.com/cloudfoundry/socks5-proxy"
	"github.com/cloudfoundry/socks5-proxy/fakes"
)

var _ = Describe("Socks5Proxy", func() {
	var (
		socks5Proxy *proxy.Socks5Proxy
		hostKey     *fakes.HostKey

		httpServerHostPort string
	)

	BeforeEach(func() {
		httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))
		httpServerHostPort = strings.Split(httpServer.URL, "http://")[1]

		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		Expect(err).NotTo(HaveOccurred())

		hostKey = &fakes.HostKey{}
		hostKey.GetCall.Returns.PublicKey = signer.PublicKey()

		socks5Proxy = proxy.NewSocks5Proxy(hostKey, nil, time.Millisecond) //sock5 server defaults to a stdout logger for us
	})

	AfterEach(func() {
		proxy.ResetNetListen()
	})

	Describe("Start", func() {
		Context("when empty username is given", func() {
			It("starts a proxy to the jumpbox", func() {
				serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
				err := socks5Proxy.Start("", privateKey, serverURL)
				Expect(err).NotTo(HaveOccurred())

				socks5Addr, err := socks5Proxy.Addr()
				Expect(err).NotTo(HaveOccurred())

				socks5Client, err := goproxy.SOCKS5("tcp", socks5Addr, nil, goproxy.Direct)
				Expect(err).NotTo(HaveOccurred())

				Expect(hostKey.GetCall.CallCount).To(Equal(1))
				Expect(hostKey.GetCall.Receives.Username).To(Equal("jumpbox"))
				Expect(hostKey.GetCall.Receives.PrivateKey).To(Equal(privateKey))
				Expect(hostKey.GetCall.Receives.ServerURL).To(Equal(serverURL))

				conn, err := socks5Client.Dial("tcp", httpServerHostPort)
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
				Expect(err).NotTo(HaveOccurred())
				defer conn.Close() //nolint:errcheck

				status, err := bufio.NewReader(conn).ReadString('\n') //nolint:ineffassign,staticcheck
				Expect(status).To(Equal("HTTP/1.0 200 OK\r\n"))
			})

			Context("when starting the proxy a second time", func() {
				It("no-ops on the second run", func() {
					serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
					err := socks5Proxy.Start("", privateKey, serverURL)
					Expect(err).NotTo(HaveOccurred())

					err = socks5Proxy.Start("", privateKey, serverURL)
					Expect(err).NotTo(HaveOccurred())

					socks5Addr, err := socks5Proxy.Addr()
					Expect(err).NotTo(HaveOccurred())

					socks5Client, err := goproxy.SOCKS5("tcp", socks5Addr, nil, goproxy.Direct)
					Expect(err).NotTo(HaveOccurred())

					conn, err := socks5Client.Dial("tcp", httpServerHostPort)
					Expect(err).NotTo(HaveOccurred())

					_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
					Expect(err).NotTo(HaveOccurred())
					defer conn.Close() //nolint:errcheck

					status, err := bufio.NewReader(conn).ReadString('\n') //nolint:ineffassign,staticcheck
					Expect(status).To(Equal("HTTP/1.0 200 OK\r\n"))
				})
			})

			Context("when opening the port fails", func() {
				It("returns an error", func() {
					serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
					proxy.SetNetListen(func(string, string) (net.Listener, error) {
						return nil, errors.New("coconut")
					})

					err := socks5Proxy.Start("", privateKey, serverURL)
					Expect(err).To(MatchError("open port: coconut"))
				})
			})
		})

		Context("when a custom username is given", func() {
			JustBeforeEach(func() {
				signer, err := ssh.ParsePrivateKey([]byte(privateKey))
				Expect(err).NotTo(HaveOccurred())

				hostKey = &fakes.HostKey{}
				hostKey.GetCall.Returns.PublicKey = signer.PublicKey()

				socks5Proxy = proxy.NewSocks5Proxy(hostKey, nil, time.Millisecond) //sock5 server defaults to a stdout logger for us
			})

			It("returns a dialer that proxies to the jumpbox with a custom user", func() {
				serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "custom-username")
				err := socks5Proxy.Start("custom-username", privateKey, serverURL)
				Expect(err).NotTo(HaveOccurred())

				socks5Addr, err := socks5Proxy.Addr()
				Expect(err).NotTo(HaveOccurred())

				socks5Client, err := goproxy.SOCKS5("tcp", socks5Addr, nil, goproxy.Direct)
				Expect(err).NotTo(HaveOccurred())

				Expect(hostKey.GetCall.CallCount).To(Equal(1))
				Expect(hostKey.GetCall.Receives.Username).To(Equal("custom-username"))
				Expect(hostKey.GetCall.Receives.PrivateKey).To(Equal(privateKey))
				Expect(hostKey.GetCall.Receives.ServerURL).To(Equal(serverURL))

				conn, err := socks5Client.Dial("tcp", httpServerHostPort)
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
				Expect(err).NotTo(HaveOccurred())
				defer conn.Close() //nolint:errcheck

				status, err := bufio.NewReader(conn).ReadString('\n') //nolint:ineffassign,staticcheck
				Expect(status).To(Equal("HTTP/1.0 200 OK\r\n"))
			})
		})
	})

	Describe("Dialer", func() {
		Context("when empty username is given", func() {
			It("returns a dialer that proxies to the jumpbox with user 'jumpbox'", func() {
				serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
				dialer, err := socks5Proxy.Dialer("", privateKey, serverURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(hostKey.GetCall.CallCount).To(Equal(1))
				Expect(hostKey.GetCall.Receives.Username).To(Equal("jumpbox"))
				Expect(hostKey.GetCall.Receives.PrivateKey).To(Equal(privateKey))
				Expect(hostKey.GetCall.Receives.ServerURL).To(Equal(serverURL))

				conn, err := dialer("tcp", httpServerHostPort)
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
				Expect(err).NotTo(HaveOccurred())
				defer conn.Close() //nolint:errcheck

				status, err := bufio.NewReader(conn).ReadString('\n') //nolint:ineffassign,staticcheck
				Expect(status).To(Equal("HTTP/1.0 200 OK\r\n"))
			})

			Context("failure cases", func() {
				Context("when it cannot parse the private key", func() {
					It("returns an error", func() {
						serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
						_, err := socks5Proxy.Dialer("", "some-bad-private-key", serverURL)
						Expect(err).To(MatchError("parse private key: ssh: no key found"))
					})
				})

				Context("when it cannot get the host key", func() {
					BeforeEach(func() {
						hostKey.GetCall.Returns.Error = errors.New("banana")
					})

					It("returns an error", func() {
						serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
						_, err := socks5Proxy.Dialer("", privateKey, serverURL)
						Expect(err).To(MatchError("get host key: banana"))
					})
				})

				Context("when it cannot dial the jumpbox url", func() {
					It("returns an error", func() {
						_, err := socks5Proxy.Dialer("", privateKey, "some-bad-url")
						Expect(err).To(MatchError("ssh dial: dial tcp: address some-bad-url: missing port in address"))
					})
				})

			})
		})

		Context("when a custom username is given", func() {
			JustBeforeEach(func() {
				signer, err := ssh.ParsePrivateKey([]byte(privateKey))
				Expect(err).NotTo(HaveOccurred())

				hostKey = &fakes.HostKey{}
				hostKey.GetCall.Returns.PublicKey = signer.PublicKey()

				socks5Proxy = proxy.NewSocks5Proxy(hostKey, nil, time.Millisecond) //sock5 server defaults to a stdout logger for us
			})

			It("returns a dialer that proxies to the jumpbox with a custom user", func() {
				serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "custom-username")
				dialer, err := socks5Proxy.Dialer("custom-username", privateKey, serverURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(hostKey.GetCall.CallCount).To(Equal(1))
				Expect(hostKey.GetCall.Receives.Username).To(Equal("custom-username"))
				Expect(hostKey.GetCall.Receives.PrivateKey).To(Equal(privateKey))
				Expect(hostKey.GetCall.Receives.ServerURL).To(Equal(serverURL))

				conn, err := dialer("tcp", httpServerHostPort)
				Expect(err).NotTo(HaveOccurred())

				_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
				Expect(err).NotTo(HaveOccurred())
				defer conn.Close() //nolint:errcheck

				status, err := bufio.NewReader(conn).ReadString('\n') //nolint:ineffassign,staticcheck
				Expect(status).To(Equal("HTTP/1.0 200 OK\r\n"))
			})
		})
	})

	Describe("Addr", func() {
		Context("when the proxy has been started", func() {
			BeforeEach(func() {
			})

			It("returns a valid address of the socks5 proxy", func() {
				serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
				err := socks5Proxy.Start("", privateKey, serverURL)
				Expect(err).NotTo(HaveOccurred())

				addr, err := socks5Proxy.Addr()
				Expect(err).NotTo(HaveOccurred())
				Expect(addr).To(MatchRegexp("127.0.0.1:\\d+"))
			})
		})

		Context("when no proxy has been started", func() {
			It("returns an error", func() {
				_, err := socks5Proxy.Addr()
				Expect(err).To(MatchError("socks5 proxy is not running"))
			})
		})
	})

	Describe("SetListenPort", func() {
		It("listens on the provided port", func() {
			serverURL := proxy.StartTestSSHServer(httpServerHostPort, privateKey, "")
			socks5Proxy.SetListenPort(1337)
			err := socks5Proxy.Start("", privateKey, serverURL)
			Expect(err).NotTo(HaveOccurred())

			socks5Addr, err := socks5Proxy.Addr()
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Contains(socks5Addr, ":1337")).To(Equal(true))

			socks5Client, err := goproxy.SOCKS5("tcp", socks5Addr, nil, goproxy.Direct)
			Expect(err).NotTo(HaveOccurred())

			Expect(hostKey.GetCall.CallCount).To(Equal(1))
			Expect(hostKey.GetCall.Receives.Username).To(Equal("jumpbox"))
			Expect(hostKey.GetCall.Receives.PrivateKey).To(Equal(privateKey))
			Expect(hostKey.GetCall.Receives.ServerURL).To(Equal(serverURL))

			conn, err := socks5Client.Dial("tcp", httpServerHostPort)
			Expect(err).NotTo(HaveOccurred())

			_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close() //nolint:errcheck

			status, err := bufio.NewReader(conn).ReadString('\n') //nolint:ineffassign,staticcheck
			Expect(status).To(Equal("HTTP/1.0 200 OK\r\n"))
		})
	})
})
