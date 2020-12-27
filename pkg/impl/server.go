package impl

import (
	"crypto/tls"
	"github.com/Qv2ray/gun/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"net"
)

type GunServiceServerImpl struct {
	RemoteAddr string
	LocalAddr  string
	CertPath   string
	KeyPath    string
	Cleartext  bool
}

func (g GunServiceServerImpl) Run() {
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

	proto.RegisterGunServiceServer(s, GunServiceServerImpl{
		RemoteAddr: g.RemoteAddr,
	})

	// listen local
	listener, e := net.Listen("tcp", g.LocalAddr)
	if e != nil {
		log.Fatalf("failed to listen: %v", e)
	}

	log.Printf("starting listening on: %v", g.LocalAddr)
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
