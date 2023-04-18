package common

import (
	"context"
	"net"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

func NewDialContext(socks5 string) (DialContext, error) {
	baseDialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}
	if socks5 == "" {
		return baseDialer.DialContext, nil
	}

	// split socks5 proxy string [username:password@]host:port
	var auth *proxy.Auth = nil

	if strings.Contains(socks5, "@") {
		proxyInfo := strings.SplitN(socks5, "@", 2)
		proxyUser := strings.Split(proxyInfo[0], ":")
		if len(proxyUser) == 2 {
			auth = &proxy.Auth{
				User:     proxyUser[0],
				Password: proxyUser[1],
			}
		}
		socks5 = proxyInfo[1]
	}

	dialSocksProxy, err := proxy.SOCKS5("tcp", socks5, auth, baseDialer)
	if err != nil {
		return nil, err
	}

	contextDialer, ok := dialSocksProxy.(proxy.ContextDialer)
	if !ok {
		return nil, err
	}

	return contextDialer.DialContext, nil
}
