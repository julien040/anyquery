package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/fatedier/frp/client"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

const serverAddr = "eu-central-1-reverse.anyquery.xyz"
const serverPort = 7000
const reverseProxyDomain = "reverse.anyquery.xyz"

type Tunnel struct {
	// The tunnel ID to connect to
	ID string
	// The unhashed auth token
	AuthToken string
	// The host of the server the tunnel is connected to
	Host string
	// The port of the server the tunnel is connected to
	Port int

	// Internal fields
	srv *client.Service
}

func hashToken(token string) string {
	summed := sha256.Sum256([]byte(token))
	return hex.EncodeToString(summed[:])
}

// Create a tunnel from retrieved data
func NewTunnel(id, authToken, host string, port int) *Tunnel {
	return &Tunnel{
		ID:        id,
		AuthToken: hashToken(authToken),
		Host:      host,
		Port:      port,
	}
}

func (t *Tunnel) Connect() error {
	failExit := false
	opts := client.ServiceOptions{
		Common: &v1.ClientCommonConfig{
			ServerAddr:    serverAddr,
			ServerPort:    serverPort,
			User:          t.ID,
			LoginFailExit: &failExit,
			Transport: v1.ClientTransportConfig{
				Protocol:            "tcp",
				DialServerTimeout:   10,
				DialServerKeepAlive: 7200,
				PoolCount:           1,
			},
			Auth: v1.AuthClientConfig{
				Method: "token",
			},
		},
		ProxyCfgs: []v1.ProxyConfigurer{
			&v1.HTTPProxyConfig{
				ProxyBaseConfig: v1.ProxyBaseConfig{
					Name: t.ID,
					Type: "http",
					Metadatas: map[string]string{
						"auth_token": t.AuthToken,
					},
					ProxyBackend: v1.ProxyBackend{
						LocalIP:   t.Host,
						LocalPort: t.Port,
					},
					// If you uncomment these lines, the tunnel won't work, and will create unvalid HTTP requests
					/* Transport: v1.ProxyTransport{
						ProxyProtocolVersion: "v1",
						BandwidthLimitMode:   "client",
					}, */
				},
				DomainConfig: v1.DomainConfig{
					SubDomain:     t.ID,
					CustomDomains: nil,
				},
			},
		},
		VisitorCfgs: []v1.VisitorConfigurer{},
	}

	// Connect to the server
	srv, err := client.NewService(opts)
	if err != nil {
		return fmt.Errorf("error creating service: %w", err)
	}

	t.srv = srv

	return t.srv.Run(context.Background())
}

// Close the tunnel
func (t *Tunnel) Close() {
	t.srv.Close()
}
