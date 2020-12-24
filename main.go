//go:generate protoc gun.proto --go_out=plugins=grpc:.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	RunMode    = flag.String("mode", "", "run mode. must be client or server")
	LocalAddr  = flag.String("local", "", "local address to listen")
	RemoteAddr = flag.String("remote", "", "remote address to connect")
	CertPath   = flag.String("cert", "", "(server) certificate (*.pem) path")
	KeyPath    = flag.String("key", "", "(server) certificate key (*.key) path")
	ServerName = flag.String("sni", "", "(client) optionally override SNI")
)

func main() {
	options := map[string]string{}
	_, sip003 := os.LookupEnv("SS_LOCAL_HOST")
	if sip003 {
		log.Println("start as SIP003 plugin, command line parameter is ignored.")
		*LocalAddr = fmt.Sprintf("%s:%s", os.Getenv("SS_LOCAL_HOST"), os.Getenv("SS_LOCAL_PORT"))
		*RemoteAddr = fmt.Sprintf("%s:%s", os.Getenv("SS_REMOTE_HOST"), os.Getenv("SS_REMOTE_PORT"))
		optionArr := strings.Split(os.Getenv("SS_PLUGIN_OPTIONS"), ";")
		if optionArr[0] != "server" {
			*RunMode = "client"
		} else {
			*RunMode = "server"
		}
		optionArr = optionArr[1:]
		for _, s := range optionArr {
			kv := strings.Split(s, "=")
			if len(kv) != 2 {
				log.Println(s)
				log.Fatalln("Can't parse plugin option")
			}
			options[kv[0]] = options[kv[1]]
		}

		*ServerName = readOption(options, "sni")
		*CertPath = readOption(options, "cert")
		*KeyPath = readOption(options, "key")
	} else {
		flag.Parse()
	}

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

func readOption(options map[string]string, key string) string {
	val, ok := options[key]
	if !ok {
		val = ""
	}
	return val
}
