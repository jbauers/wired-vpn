<img height="64" src="./docs/wired.png">

# Wired

This is a VPN server and client using [WireGuard](https://www.wireguard.com) with authentication using OIDC. It is experimental, a playground and learning experience, and **should not be used in a production environment**, at least not in its current state. There are 3 components that work together:

- [client](./client)
- [server/proxy](./server/proxy)
- [server/api](./server/api)

This project is not affiliated with the WireGuard project. WireGuard is a registered trademark of Jason A. Donenfeld.

### What?

The **client** is a cross-platform GUI application written in Go, using Fyne. The idea is to only give end users a `Connect` button. Non-technical people do not care about keys or IPs, and the less options, the more security (or something like that).

The **server** consist of an "API" backend and a proxy. The proxy is a very simple Openresty (Nginx/Lua) web server for OIDC auth. The API is a Go application that stores things in Redis and configures interfaces.

#### Considerations

- This was made with non-technical end users in mind. Everyone can click a `Connect` button, not everyone is comfortable with configuration files, let alone configuring tunnels over a CLI.
- This was made with the needs of organisations in mind. You may ask why email addresses are stored, and not only keys. We want to know who's on our network, and this is not a privacy preserving VPN.
- This was made with Cloud in mind. You can absolutely run this on your own boxes, but the reason for chosing Redis, for example, is that we can buy a managed Redis for "state", dump the rest on a container orchestration platform, and go crazy with ACLs on our network.

### How?

Upon `Connect`, the client initiates an auth flow with our `server/proxy` and starts listening on a local callback URL. Once the user is authenticated, the proxy passes the request downstream to the `server/api` backend, which handles this client. No private keys are ever exchanged - the client simply adds its public key to the inital request, which is forwarded to the backend after successful authentication. The backend then adds this WireGuard peer, and replies with its own public key to the callback URL. Once the client receives the callback, it configures its own WireGuard interface.

- Authentication happens between our proxy and the identity provider. See [auth.lua](./server/proxy/auth.lua).
- The API expects security aspects are taken care of upstream. It sanity-checks only the provided public key, as this is coming directly from the client. User and group are added as headers by our proxy during the auth flow, and aren't configurable by end users.
- Keys can be rotated freely and the API is "smart" enough to account for that. When the client application starts, it generates a new private key. Connecting will send the new public key to the API, which will rotate the peer on its side if the public key differs from the stored one. Keys are also expired server-side, and this expiration is configurable. Reasonable is probably something like 12h for a working day plus padding. If the key is removed server-side, the client will lose connection. When reconnecting, a new PSK is then used - either with a new client public key or not.

## Usage

Have a look at [server/example.settings.json](./server/example.settings.json) first. For the OIDC endpoint, only `https` is allowed and automatically added. Get the OIDC `client_id` and `client_secret`, as well as the `discovery_url` from the IdP. You will need to add a redirect URI on the IdP side, which will be the `http_endpoint`  as configured in the settings, plus proto and path `/redirect_uri`: `https://example.com/redirect_uri`. While testing locally, you should still set this, and add an `/etc/hosts` entry on your machine. A script to generate self-signed SSL certificates is included.

Also have a look at [server/docker-compose.yml](./server/docker-compose.yml). Once everything is configured:

- [./run_server.sh](./run_server.sh) to run the proxy and backend
- [./run_client.sh](./run_client.sh) to build and run the client

There's a `LOCAL=true` environment variable for the backend that when set will add the private IP of your Docker container as the endpoint. When running everything locally, you will then connect into the API backend container over VPN. This is useful for testing, but will allow bypassing the proxy: With the default settings, you can for example `curl 10.100.0.1:9000`, which will work, whereas `curl localhost:9000` is not exposed.

### Current state and future plans
- This has really only been tested on Linux, with OneLogin as the IdP. It *should* be easy enough to add further support, both in terms of cross-platform and multiple IdPs (OIDC is a standard after all).
- Bit rough around the edges. The server is in a fairly good state, but especially the client will need more care.
- Firewall is something that'd be nice to configure automatically.
- Tests. Probably a good idea.
- Something that may be really interesting, but will add complexity, is splitting the backend API and the actual WireGuard servers by adding a WSS message queue for communication. Mullvad's [wg-manager](https://github.com/mullvad/wg-manager) does something like this. You would have the client interacting with the central control plane consisting of the proxy and the API during auth, with the API passing commands to add/delete peers over the message queue to the servers listening. Now the servers themselves can be fairly simple and spread across your network(s), and the clients would connect to an endpoint that can live somewhere entirely different than the control plane.

