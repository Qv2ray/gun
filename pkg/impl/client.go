package impl

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Qv2ray/gun/pkg/cert"
	"github.com/Qv2ray/gun/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type GunServiceClientImpl struct {
	RemoteAddr  string
	LocalAddr   string
	ServerName  string
	Cleartext   bool
	UdpSessions *sync.Map
}

type ClientUdpSession struct {
	LastActive time.Time
	Tun        proto.GunService_TunDatagramClient
}

func (g GunServiceClientImpl) Run() {
	g.UdpSessions = new(sync.Map)

	// start TCP local
	local, err := net.Listen("tcp", g.LocalAddr)
	if err != nil {
		log.Fatalf("failed to listen local: %v", err)
	}
	log.Printf("client listening tcp at %v", g.LocalAddr)

	// start UDP local
	localUdp, err := net.ListenPacket("udp", g.LocalAddr)
	if err != nil {
		log.Fatalf("failed to listen udp local: %v", err)
	}
	log.Printf("client listening udp at %v", g.LocalAddr)

	// select h2/h2c
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

	// dial
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
	// work loops
	go g.tcpLoop(local, client)
	go g.scanInactiveSession(2 * time.Minute)
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
			defer accept.Close()

			// connect rpc
			tun, err := client.Tun(context.Background())
			if err != nil {
				log.Printf("failed to create context: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// down link
			go func() {
				defer wg.Done()
				for {
					recv, err := tun.Recv()
					if err != nil {
						if status.Code(err) != codes.Unavailable && status.Code(err) != codes.OutOfRange {
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

			// up link
			go func() {
				defer wg.Done()
				buf := make([]byte, 32768)
				for {
					nRecv, err := accept.Read(buf)
					if err != nil {
						if status.Code(err) != codes.Unavailable && status.Code(err) != codes.OutOfRange {
							log.Printf("local read conn closed: %v", err)
						}
						if err = tun.CloseSend(); err != nil {
							log.Printf("remote close uplink conn fail: %v", err)
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

			wg.Wait()
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
		addrStr := addr.String()

		// associate to exist session
		var session ClientUdpSession
		s, sessionReused := g.UdpSessions.Load(addrStr)
		if !sessionReused {
			// not exist, init new session
			t, err := client.TunDatagram(context.Background())
			if err != nil {
				log.Printf("failed to create context: %v", err)
				return
			}

			session = ClientUdpSession{
				LastActive: time.Now(),
				Tun:        t,
			}
			g.UdpSessions.Store(addrStr, session)
			log.Printf("readfrom: %v <-> %v", local.LocalAddr(), addr)
		} else {
			session = s.(ClientUdpSession)
		}

		tun := session.Tun
		err = tun.Send(&proto.Hunk{Data: buf[:l]})
		if err != nil {
			log.Printf("remote write packet conn closed: %v", err)
			continue
		}
		session.LastActive = time.Now()

		if sessionReused {
			// there's already a udp down link goroutine, let it handle down link
			continue
		}

		// down link
		go func() {
			for {
				recv, err := tun.Recv()
				if err != nil {
					if status.Code(err) != codes.Unavailable && status.Code(err) != codes.OutOfRange {
						log.Printf("remote read packet conn closed: %v", err)
					}
					// when error, it's obvious
					// when eof, it means server timed out, new assoc needed
					g.clearUdpSession(addrStr)
					return
				}
				_, err = local.WriteTo(recv.Data, addr)
				if err != nil {
					log.Printf("local write packet conn closed: %v", err)
					return
				}
				session.LastActive = time.Now()
			}
		}()
	}
}

func (g GunServiceClientImpl) scanInactiveSession(timeout time.Duration) {
	tick := time.NewTicker(timeout)
	for {
		<-tick.C
		needClear := make([]string, 0)
		now := time.Now()
		g.UdpSessions.Range(func(key, value interface{}) bool {
			session := value.(ClientUdpSession)
			if session.LastActive.Add(timeout).Before(now) {
				needClear = append(needClear, key.(string))
			}
			return true
		})

		for _, k := range needClear {
			g.clearUdpSession(k)
		}
	}
}

func (g GunServiceClientImpl) clearUdpSession(name string) {
	s, ok := g.UdpSessions.Load(name)
	if !ok {
		return
	}
	log.Printf("clear udp session %v", name)
	session := s.(ClientUdpSession)
	e := session.Tun.CloseSend()
	if e != nil {
		log.Printf("error when clear session %v, %v", name, e)
	}
	g.UdpSessions.Delete(name)
}
