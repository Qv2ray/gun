package impl

import (
	"context"
	"errors"
	"github.com/Qv2ray/gun/pkg/cert"
	"io"
	"log"
	"net"
	"time"

	"github.com/Qv2ray/gun/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
)

type GunServiceClientImpl struct {
	RemoteAddr string
	LocalAddr  string
	ServerName string
	Nat        NatTable
}

type NatTable map[string]proto.GunService_TunDatagramClient

func (g GunServiceClientImpl) Run() {
	g.Nat = make(NatTable)
	// start TCP local
	local, err := net.Listen("tcp", g.LocalAddr)
	if err != nil {
		log.Fatalf("failed to listen local: %v", err)
	}

	log.Printf("client listening at %v", g.LocalAddr)
	localUdp, err := net.ListenPacket("udp", g.LocalAddr)
	if err != nil {
		log.Fatalf("failed to listen udp local: %v", err)
	}

	log.Printf("client listening at %v", g.LocalAddr)

	roots, err := cert.GetSystemCertPool()
	if err != nil {
		log.Fatalf("failed to get system certificate pool")
	}
	conn, err := grpc.Dial(
		g.RemoteAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(roots, g.ServerName)),
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
	// TCP work loop
	go g.tcpLoop(local, client)
	g.udpLoop(localUdp, client)
}

func (g GunServiceClientImpl) tcpLoop(local net.Listener, client proto.GunServiceClient) {
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

func (g GunServiceClientImpl) udpLoop(local net.PacketConn, client proto.GunServiceClient) {
	for {
		buf := make([]byte, 32768)
		l, addr, err := local.ReadFrom(buf)
		if err != nil {
			log.Printf("failed to read udp packet: %v", err)
		}
		addrStr :=addr.String()
		// associate to exist tun
		tun, ok := g.Nat[addrStr]
		if !ok {
			tun, err = client.TunDatagram(context.Background())
			if err != nil {
				log.Printf("failed to create context: %v", err)
				return
			}
			g.Nat[addrStr] = tun
		}
		err = tun.Send(&proto.Hunk{Data: buf[:l]})
		if err != nil {
			log.Printf("remote write packet conn closed: %v", err)
			return
		}
		go func() {
			for {
				recv, err := tun.Recv()
				if err != nil {
					if !errors.Is(err, io.EOF) {
						log.Printf("remote read packet conn closed: %v", err)
					}
					return
				}
				_, err = local.WriteTo(recv.Data, addr)
				if err != nil {
					log.Printf("local write packet conn closed: %v", err)
					return
				}
			}
		}()
	}
}
