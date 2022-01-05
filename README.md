<img height="64" src="./docs/wired.png">

# Wired

This is a VPN server and client using [WireGuard](https://www.wireguard.com) with authentication using OIDC. It is experimental, a playground and learning experience, and **should not be used in a production environment**, at least not in its current state. There are 4 components that work together:

- [server/auth](./server/auth): Public-facing OIDC auth proxy.
- [server/control](./server/control): Private control plane consisting of a backend, [Mullvad's message queue](https://github.com/mullvad/message-queue) and Redis.
- [server/vpn](./server/vpn): Public-facing WireGuard server.
- [client](./client): Cross-platform client GUI application. **Here be dragons!**

This project is not affiliated with the WireGuard project. WireGuard is a registered trademark of Jason A. Donenfeld.

## TL;DR

1. `./run_server.sh`
2. `./test.sh`
3. Look at some logs and read some code.

### What?

The **client** is a cross-platform GUI application written in Go, using Fyne. The idea is to only give end users a `Connect` button. Non-technical people do not care about keys or IPs, and the less options, the more security (or something like that).

The **server** consist of a control plane, a proxy and any number of WireGuard servers connected to the control plane. The proxy is a very simple Openresty (Nginx/Lua) web server for OIDC auth. The control plane is a Go application that stores things in Redis and publishes ADD or DEL messages over Websocket to connected WireGuard servers, which then configure their interfaces for clients to connect.

#### Considerations

- This was made with non-technical end users in mind. Everyone can click a `Connect` button, not everyone is comfortable with configuration files, let alone configuring tunnels over a CLI.
- This was made with the needs of organisations in mind. You may ask why email addresses are stored, and not only keys. We want to know who's on our network, and this is not a privacy preserving VPN.
- This was made with Cloud in mind. You can absolutely run this on your own boxes, but the reason for chosing Redis, for example, is that we can buy a managed Redis for "state", dump the rest on a container orchestration platform, and go crazy with ACLs on our network.

### How?

On startup, the WireGuard servers will POST their configuration - endpoint, public key and port for example - to the control plane, which will store this information in Redis. They will then start listening on the WebSocket address of the control plane for ADD or DEL messages regarding peers. When they receive a message, they will immediately update their WireGuard interface to allow or disallow a connection.

Upon `Connect`, the client initiates an auth flow with our `server/auth` and starts listening on a local callback URL. Once the user is authenticated, the proxy passes the request downstream to the `server/control` control plane. No private keys are ever exchanged - the client simply adds its public key to the inital request, which is forwarded to the control plane after successful authentication. The control plane then publishes a message to connected WireGuard servers over Websocket, and replies with the public key for this WireGuard server on the callback URL. Once the client receives the callback, it configures its own WireGuard interface and can connect to the WireGuard server it was assigned to from the control plane.

- Authentication happens between our proxy and the identity provider. See [auth.lua](./server/auth/auth.lua).
- The control plane expects security aspects are taken care of upstream. It sanity-checks only the provided public key, as this is coming directly from the client. User and group are added as headers by our proxy during the auth flow, and aren't configurable by end users.
- Keys can be rotated freely and the control plane is "smart" enough to account for that. When the client application starts, it generates a new private key. Connecting will send the new public key to the API, which will rotate the peer on its side if the public key differs from the stored one. Keys are also expired server-side, and this expiration is configurable. Reasonable is probably something like 12h for a working day plus padding. If the key is removed server-side, the client will lose connection. When reconnecting, a new PSK is then used - either with a new client public key or not.

### Why?

I liked the idea of a single source of truth, but the ability to scatter my WireGuard servers across the network. The control plane does not care about the public endpoints  or keys of its WireGuard servers, as long as it knows about the server names that are connecting, those servers can change their configuration. You can then have different firewall rules for different servers and place them on different subnets to create something that's not really a mesh, but also not really a traditional hub and spoke setup (although it is more of the latter).

## Usage

Have a look at [server/example.settings.json](./server/example.settings.json) first. For the OIDC endpoint, only `https` is allowed and automatically added. Get the OIDC `client_id` and `client_secret`, as well as the `discovery_url` from the IdP. You will need to add a redirect URI on the IdP side, which will be the `http_endpoint`  as configured in the settings, plus proto and path `/redirect_uri`: `https://example.com/redirect_uri`. While testing locally, you should still set this, and add an `/etc/hosts` entry on your machine. A script to generate self-signed SSL certificates is included.

Also have a look at [server/docker-compose.yml](./server/docker-compose.yml). Once everything is configured:

- [./run_server.sh](./run_server.sh) to run the proxy and control plane
- [./run_client.sh](./run_client.sh) to build and run the client
- [./test.sh](./test.sh) to do some simple sanity checking (bypasses auth, doesn't need a client).

### Current state and future plans
- Has really only been tested on Linux, with OneLogin as the IdP. It *should* be easy enough to add further support, both in terms of cross-platform and multiple IdPs (OIDC is a standard after all).
- Cleanup and docs. There's still some leftovers and missing clarification.
- Security should be okay as long as you set up your network okay, but I'm sure it's far from perfect.
- Reliability - just panics everywhere during debugging, some nicer error handling could be good.
- The client application is a bit of a dumpster fire.
- Firewall is something that'd be nice to configure automatically.
- Tests. Add more and better ones.
