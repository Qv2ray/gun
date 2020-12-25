# gun
Grpc tUNnel over Cloudflare

## Headline
```
发一个自用了一个晚上的代理。
程序本身很简单，用来过 CloudFlare 足够了。
```

## Example
### Server
1. Go to your domain in CloudFlare. In "Network" tab, turn on gRPC.

2. In "SSL/TLS" tab, choose "Source Servers" subtab. Create a new certificate and save as `cert.pem` and `cert.key`.

3. In "DNS" Tab, add a record pointing to your own server. Make sure the proxy state is "Proxied".

4. Run and persist this on server. This example will forward the inbound traffic to `127.0.0.1:8899`.
```bash
gun -mode server -local :443 -remote 127.0.0.1:8899 -cert cert.pem -key cert.key
```

### Client
1. Assume the domain of server is `grpc.example.com`.

2. Run locally and persist. This will tunnel connections from `127.0.0.1:8899` to remote.
```bash
gun -mode client -local 127.0.0.1:8899 -remote grpc.example.com:443
```

## License
AGPL3
