package main

import (
	"errors"
	"net"
	"os"
)

type SIP003Arguments struct {
	LocalAddr  string
	RemoteAddr string
	Options    string
}

func GetSIP003Arguments() (arguments *SIP003Arguments, err error) {
	localHost, exists := os.LookupEnv("SS_LOCAL_HOST")
	if !exists {
		return nil, errors.New("no local host")
	}
	localPort, exists := os.LookupEnv("SS_LOCAL_PORT")
	if !exists {
		return nil, errors.New("no local port")
	}
	remoteHost, exists := os.LookupEnv("SS_REMOTE_HOST")
	if !exists {
		return nil, errors.New("no remote host")
	}
	remotePort, exists := os.LookupEnv("SS_REMOTE_PORT")
	if !exists {
		return nil, errors.New("no remote port")
	}
	options, exists := os.LookupEnv("SS_PLUGIN_OPTIONS")
	if !exists {
		return nil, errors.New("no plugin options")
	}
	return &SIP003Arguments{
		LocalAddr:  net.JoinHostPort(localHost, localPort),
		RemoteAddr: net.JoinHostPort(remoteHost, remotePort),
		Options:    options,
	}, nil
}
