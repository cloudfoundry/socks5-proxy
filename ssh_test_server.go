package proxy

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type current struct {
	t        time.Time
	interval time.Duration
	end      chan bool
}

func StartTestSSHServer(httpServerURL, sshPrivateKey, userName string, interval time.Duration) string {
	if userName == "" {
		userName = "jumpbox"
	}

	signer, err := ssh.ParsePrivateKey([]byte(sshPrivateKey))
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if c.User() != userName {
				return nil, fmt.Errorf("unknown user: %q", c.User())
			}

			if string(signer.PublicKey().Marshal()) == string(pubKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}

	end := make(chan bool)

	checker := &current{
		t:        time.Now(),
		interval: interval,
		end:      end,
	}

	go func() {
		for {
			fmt.Println("wassssup")
			select {
			case <-end:
				listener.Close()
				fmt.Println("he gone")
				return
			default:
				nConn, err := listener.Accept()
				if err != nil {
					log.Fatal("failed to accept incoming connection: ", err)
				}

				serverConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
				if err != nil {
					log.Println("failed to handshake: ", err)
					return
				}

				go checker.checkReqs(reqs)

				fmt.Println("he here")
				handle(chans, httpServerURL, checker, serverConn)
			}
		}
	}()

	return listener.Addr().String()
}

func handle(chans <-chan ssh.NewChannel, httpServerURL string, checker *current, serverConn *ssh.ServerConn) {
	time.Sleep(1 * time.Second)

	for newChannel := range chans {
		t := time.Now()
		if t.After(checker.t.Add(checker.interval)) {
			fmt.Println("yolo")
			serverConn.Close()
			close(checker.end)
			return
		}
		if newChannel.ChannelType() != "direct-tcpip" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, _, err := newChannel.Accept()
		if err != nil {
			log.Fatalf("Could not accept channel: %v", err)
		}
		defer channel.Close()

		data, err := bufio.NewReader(channel).ReadString('\n')
		if err != nil {
			log.Fatalf("Can't read data from channel: %v", err)
		}

		httpConn, err := net.Dial("tcp", httpServerURL)
		if err != nil {
			log.Fatalf("Could not open connection to http server: %v", err)
		}
		defer httpConn.Close()

		_, err = httpConn.Write([]byte(data + "\r\n\r\n"))
		if err != nil {
			log.Fatalf("Could not write to http server: %v", err)
		}

		data, err = bufio.NewReader(httpConn).ReadString('\n')
		if err != nil {
			log.Fatalf("Can't read data from http conn: %v", err)
		}

		_, err = channel.Write([]byte(data))
		if err != nil {
			log.Fatalf("Can't write data to channel: %v", err)
		}
	}
}

func (c current) checkReqs(reqs <-chan *ssh.Request) {
	for req := range reqs {
		if req.Type == "bosh-cli-keep-alive@bosh.io" {
			c.t = time.Now()
		}

		if req.WantReply {
			req.Reply(false, nil)
		}
	}
}
