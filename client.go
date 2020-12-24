package main

import (
	"context"
	"crypto/x509"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net"
)

type GunServiceClientImpl struct {
	RemoteAddr string
	LocalAddr string
	ServerName string
}

func (g GunServiceClientImpl) Run() {
	local, err := net.Listen("tcp", g.LocalAddr)
	if err != nil {
		log.Fatalf("failed to listen local: %v", err)
	}

	log.Printf("client listening at %v", g.LocalAddr)

	roots, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("failed to get system certificate pool")
	}
	conn, err := grpc.Dial(g.RemoteAddr, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(roots, g.ServerName)))
	if err != nil {
		log.Fatalf("failed to dial remote: %v", err)
	}

	client := NewGunServiceClient(conn)
	for {
		accept, err := local.Accept()
		if err != nil {
			continue
		}

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
						//log.Printf("failed to recv from remote: %v", err)
						return
					}
					_, err = accept.Write(recv.Data)
					if err != nil {
						//log.Printf("failed to write to conn: %v", err)
						return
					}
				}
			}()
			buf := make([]byte, 32768)
			for {
				nRecv, err := accept.Read(buf)
				if err != nil {
					//log.Printf("failed to recv from local: %v", err)
					return
				}
				err = tun.Send(&Hunk{Data: buf[:nRecv]})
				if err != nil {
					//log.Printf("failed to send to remote: %v", err)
					return
				}
			}
		}()
	}
}