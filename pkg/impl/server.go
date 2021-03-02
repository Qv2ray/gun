package impl

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Qv2ray/gun/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type GunServiceServerImpl struct {
	RemoteAddr  string
	LocalAddr   string
	CertPath    string
	KeyPath     string
	Cleartext   bool
	UdpSessions *sync.Map

	ServiceName string
}

func (g GunServiceServerImpl) Run() {
	g.UdpSessions = new(sync.Map)
	var s *grpc.Server
	if !g.Cleartext {
		pub, err := ioutil.ReadFile(g.CertPath)
		if err != nil {
			log.Fatalf("failed to read certificate: %v", err)
		}
		key, err := ioutil.ReadFile(g.KeyPath)
		if err != nil {
			log.Fatalf("failed to read certificate key: %v", err)
		}
		cert, e := tls.X509KeyPair(pub, key)
		if e != nil {
			log.Fatalf("failed to build certificate pair: %v", e)
		}
		log.Println("certificate pair built successfully")
		s = grpc.NewServer(grpc.Creds(credentials.NewServerTLSFromCert(&cert)))
	} else {
		s = grpc.NewServer()
	}

	proto.RegisterGunServiceServerX(s, g, g.ServiceName)

	// listen local
	listener, e := net.Listen("tcp", g.LocalAddr)
	if e != nil {
		log.Fatalf("failed to listen: %v", e)
	}

	log.Printf("starting listening on: %v", g.LocalAddr)
	go g.scanInactiveSession(2 * time.Minute)
	e = s.Serve(listener)
	log.Fatalf("server abort: %v", e)
}

func (g GunServiceServerImpl) Tun(server proto.GunService_TunServer) error {
	conn, err := net.Dial("tcp", g.RemoteAddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	errChan := make(chan error)

	go func() {
		for {
			if recv, err := server.Recv(); err != nil {
				errChan <- err
				return
			} else if _, err = conn.Write(recv.Data); err != nil {
				errChan <- err
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 32768)
		for {
			if nRecv, err := conn.Read(buf); err != nil {
				errChan <- err
				return
			} else if err = server.Send(&proto.Hunk{Data: buf[:nRecv]}); err != nil {
				errChan <- err
				return
			}
		}
	}()

	err = <-errChan
	return err
}

type ServerUdpSession struct {
	LastActive time.Time
	Tun        proto.GunService_TunDatagramServer
	Socket     net.PacketConn
}

func (g GunServiceServerImpl) TunDatagram(server proto.GunService_TunDatagramServer) error {
	raddr, err := net.ResolveUDPAddr("udp", g.RemoteAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return err
	}
	log.Printf("start new udp session %v <-> %v", conn.LocalAddr(), g.RemoteAddr)
	sessionName := conn.LocalAddr().String()
	session := ServerUdpSession{
		LastActive: time.Now(),
		Tun:        server,
		Socket:     conn,
	}
	g.UdpSessions.Store(sessionName, session)

	defer g.clearUdpSession(sessionName)

	errChan := make(chan error)

	var wg sync.WaitGroup
	wg.Add(2)

	// up link
	go func() {
		defer wg.Done()
		for {
			if recv, err := server.Recv(); err != nil {
				if status.Code(err) != codes.Unavailable && status.Code(err) != codes.OutOfRange {
					// report only when not eof, eof is not error
					errChan <- err
				} else {
					errChan <- nil
				}
				return
			} else if _, err = conn.WriteTo(recv.Data, raddr); err != nil {
				errChan <- err
				return
			}
			session.LastActive = time.Now()
		}
	}()
	go func() {
		defer wg.Done()
		buf := make([]byte, 32768)
		for {
			nRecv, remote, err := conn.ReadFrom(buf)
			if err != nil {
				errChan <- err
				return
			}
			if remote.String() != raddr.String() {
				continue
			}
			if err = server.Send(&proto.Hunk{Data: buf[:nRecv]}); err != nil {
				errChan <- err
				return
			}
			session.LastActive = time.Now()
		}
	}()
	// wait both direction complete
	wg.Wait()
	err = <-errChan
	return err
}

func (g GunServiceServerImpl) scanInactiveSession(timeout time.Duration) {
	tick := time.NewTicker(timeout)
	for {
		<-tick.C
		needClear := make([]string, 0)
		now := time.Now()
		g.UdpSessions.Range(func(key, value interface{}) bool {
			session := value.(ServerUdpSession)
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

func (g GunServiceServerImpl) clearUdpSession(name string) {
	s, ok := g.UdpSessions.Load(name)
	if !ok {
		return
	}
	log.Printf("clear udp session %v", name)
	session := s.(ServerUdpSession)
	e := session.Socket.Close()
	if e != nil {
		log.Printf("error when clear session %v, %v", name, e)
	}
	g.UdpSessions.Delete(name)
}
