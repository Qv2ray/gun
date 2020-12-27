package main

import (
	"flag"
	"github.com/Qv2ray/gun/pkg/impl"
	"log"
	"strings"
)

var (
	RunMode    = flag.String("mode", "", "run mode. must be client or server")
	LocalAddr  = flag.String("local", "", "local address to listen")
	RemoteAddr = flag.String("remote", "", "remote address to connect")
	CertPath   = flag.String("cert", "", "(server) certificate (*.pem) path")
	KeyPath    = flag.String("key", "", "(server) certificate key (*.key) path")
	ServerName = flag.String("sni", "", "(client) optionally override SNI")
	Cleartext  = flag.Bool("cleartext", false, "use insecure HTTP/2 cleartext mode")
)

func init() {
	flag.Parse()
}

func main() {
	switch strings.ToLower(*RunMode) {
	case "client":
		impl.GunServiceClientImpl{
			RemoteAddr: *RemoteAddr,
			LocalAddr:  *LocalAddr,
			ServerName: *ServerName,
			Cleartext:  *Cleartext,
		}.Run()
	case "server":
		impl.GunServiceServerImpl{
			RemoteAddr: *RemoteAddr,
			LocalAddr:  *LocalAddr,
			CertPath:   *CertPath,
			KeyPath:    *KeyPath,
			Cleartext:  *Cleartext,
		}.Run()
	default:
		log.Fatalf("invalid run mode. must be client or server.")
	}
}
