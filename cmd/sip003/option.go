package main

import (
	"errors"
	"strings"
)

type PluginOptions map[string]string

// server:/path/to/cert.pem:/path/to/cert.key
// server:cleartext
// client
// client:new-server-name.example.com
// client:cleartext
func ParsePluginOptions(options string) (parsedOptions PluginOptions, err error) {
	parts := strings.Split(options, ":")
	switch parts[0] {
	case "client":
		switch len(parts) {
		case 1:
			return map[string]string{
				"mode": "client",
			}, nil
		case 2:
			if parts[1] == "cleartext" {
				return map[string]string{
					"mode":      "client",
					"cleartext": "cleartext",
				}, nil
			}
			return map[string]string{
				"mode": "client",
				"sni":  parts[1],
			}, nil
		default:
			return nil, errors.New("client mode expect 0 or 1 extra arguments")
		}
	case "server":
		switch len(parts) {
		case 2:
			return map[string]string{
				"mode":      "server",
				"cleartext": "cleartext",
			}, nil
		case 3:
			return map[string]string{
				"mode": "server",
				"cert": parts[1],
				"key":  parts[2],
			}, nil
		default:
			return nil, errors.New("server mode expect 1 or 2 extra arguments")
		}
	default:
		return nil, errors.New("unknown mode")
	}
}
