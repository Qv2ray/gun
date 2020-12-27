package impl

import (
	"context"
	"errors"
	"github.com/Qv2ray/gun/pkg/cert"
	"github.com/Qv2ray/gun/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"io"
	"log"
	"net"
	"time"
)

type GunServiceClientImpl struct {
	RemoteAddr string
	LocalAddr  string
	ServerName string
	Cleartext  bool
}

func (g GunServiceClientImpl) Run() {
	local, err := net.Listen("tcp", g.LocalAddr)
	if err != nil {
		log.Fatalf("failed to listen local: %v", err)
	}

	log.Printf("client listening at %v", g.LocalAddr)
	var dialOption grpc.DialOption
	if !g.Cleartext {
		roots, err := cert.GetSystemCertPool()
		if err != nil {
			log.Fatalf("failed to get system certificate pool")
		}
		dialOption = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(roots, g.ServerName))
	} else {
		dialOption = grpc.WithInsecure()
	}
	conn, err := grpc.Dial(
		g.RemoteAddr,
		dialOption,
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  500 * time.Millisecond,
				Multiplier: 1.5,
				Jitter:     0.2,
				MaxDelay:   19 * time.Millisecond,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
	)
	if err != nil {
		log.Fatalf("failed to dial remote: %v", err)
	}

	client := proto.NewGunServiceClient(conn)
	for {
		accept, err := local.Accept()
		if err != nil {
			continue
		}

		log.Printf("accepted: %v <-> %v", accept.LocalAddr(), accept.RemoteAddr())
		go func() {
			tun, err := client.Tun(context.Background())
			if err != nil {
				log.Printf("failed to create context: %v", err)
				return
			}
			go func() {
				for {
					recv, err := tun.Recv()
					if err != nil {
						if !errors.Is(err, io.EOF) {
							log.Printf("remote read conn closed: %v", err)
						}
						return
					}
					_, err = accept.Write(recv.Data)
					if err != nil {
						log.Printf("local write conn closed: %v", err)
						return
					}
				}
			}()
			buf := make([]byte, 32768)
			for {
				nRecv, err := accept.Read(buf)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						log.Printf("local read conn closed: %v", err)
					}
					return
				}
				err = tun.Send(&proto.Hunk{Data: buf[:nRecv]})
				if err != nil {
					log.Printf("remote write conn closed: %v", err)
					return
				}
			}
		}()
	}
}
