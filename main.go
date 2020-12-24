//go:generate protoc gun.proto --go_out=plugins=grpc:.
package main

import (
	"flag"
	"log"
	"strings"
)

var (
	RunMode = flag.String("mode", "", "run mode. must be client or server")
	LocalAddr = flag.String("local", "", "local address to listen")
	RemoteAddr = flag.String("remote", "", "remote address to connect")
	CertPath = flag.String("cert", "", "(server) certificate (*.pem) path")
	KeyPath = flag.String("key", "", "(server) certificate key (*.key) path")
	ServerName = flag.String("sni", "", "(client) optionally override SNI")
)

func init() {
	flag.Parse()
}

func main() {
	switch strings.ToLower(*RunMode) {
	case "client":
		GunServiceClientImpl{
			RemoteAddr: *RemoteAddr,
			LocalAddr:  *LocalAddr,
			ServerName: *ServerName,
		}.Run()
	case "server":
		GunServiceServerImpl{
			RemoteAddr: *RemoteAddr,
			LocalAddr:  *LocalAddr,
			CertPath:   *CertPath,
			KeyPath:    *KeyPath,
		}.Run()
	default:
		log.Fatalf("invalid run mode. must be client or server.")
	}
}