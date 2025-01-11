# Tunnel server

To provide HTTP tunneling service, Anyquery uses [FRP](https://github.com/fatedier/frp) as the tunnel server. There is a sidecar container managing the FRP server. It also handles the authentication and authorization of the tunnel clients.

## `server` folder

Contains the FRP sidecar service and the configuration file.

## `client` folder

Contains the FRP code to connect to the tunnel server.
