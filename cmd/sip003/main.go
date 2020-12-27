package main

import (
	"github.com/Qv2ray/gun/pkg/impl"
	"log"
)

func main() {
	log.Println("gun is running in SIP003 mode.")
	arguments, err := GetSIP003Arguments()
	if err != nil {
		log.Fatalf("failed to parse sip003 arguments: %v", err)
	}

	options, err := ParsePluginOptions(arguments.Options)
	if err != nil {
		log.Fatalf("failed to parse plugin options: %v", err)
	}

	switch options["mode"] {
	case "client":
		impl.GunServiceClientImpl{
			RemoteAddr: arguments.RemoteAddr,
			LocalAddr:  arguments.LocalAddr,
			ServerName: options["sni"],
			Cleartext:  options["cleartext"] == "cleartext",
		}.Run()
	case "server":
		impl.GunServiceServerImpl{
			RemoteAddr: arguments.LocalAddr,
			LocalAddr:  arguments.RemoteAddr,
			CertPath:   options["cert"],
			KeyPath:    options["key"],
			Cleartext:  options["cleartext"] == "cleartext",
		}.Run()
	default:
		log.Fatalf("unknown run mode")
	}
}
