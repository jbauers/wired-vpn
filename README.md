# Wired VPN

Docker based WireGuard server with OIDC authentication and group support via different interfaces. Google SSO and OneLogin have been tested. A super small Openresty frontend, a small Golang backend, and Redis holding keys.

Settings are defined in a [settings.json](./example.json) file, which is mounted (or copied) to `/`, where both backend and frontend expect this file to be. Also see [docker-compose.yml](./docker-compose.yml).

The basic idea here is to define which OIDC group should be part of which WireGuard interface, and to limit access on the firewall for a given interface. Later, the firewall could be set up automatically from `AllowedIPs` - for now, you'll have to write your own rules (with Shorewall/iptables/...).

## Status

**Don't use in production.** That said, it'll probably work *fine*, but you should probably check the code and may have to change it. There's some FIXME's still. PR's welcome :)

# Considerations

## Simplicity

The primary goal is simplicity, or writing as little code as necessary. This is a public-facing VPN with a basic use case: You have groups of people that need access to your internal network. Based on their group, they should get access to different subnets.

It's also a learning experience, and this project should be "approachable" (I'm new to Go and a shitty programmer).

## Security

Redis holds private keys of servers and clients. That's not ideal, and the assumption is that on a network level, only the backend can talk to Redis, and only the frontend can talk to the backend. It's certainly possible to create something that generates client keys locally, authenticates via OIDC, and sends the public key to the server. It's also a bit more code. Given the previous consideration, I've opted for this approach.

The client configuration is printed on the web page, and clients will have to copy and paste the config into their WireGuard application. This is on purpose, but debatable. The reasoning behind this is that if you have something intercepting your clipboard, you probably have more serious problems - and it's better not to have the config file flying around on your machine. Again, this is debatable - you can always change the code, or create a PR with good reasoning and a better approach.

Use HTTPS. If you don't deploy this to `$CLOUD_PROVIDER`, and have them take care of SSL, check out [auto-ssl](https://github.com/auto-ssl/lua-resty-auto-ssl). This is not debatable ;)

# Usage

In `dev`, `./run.sh` (throwaway) or `docker-compose up` (persistent data).

In `prd`, make sure your network is locked down and your load balancer for the frontend does SSL termination. While the backend will add the WireGuard interfaces as per `settings.json`, don't forget to map the UDP port for each interface.

## Frontend

<b>Openresty</b> with the following modules:

- lua-resty-http
- lua-resty-session
- lua-resty-openidc

Users are first authenticated with the OpenIDC module. There's some light validation of the OIDC provider's response before the user and its group are proxy passed as headers to the backend. There's not much on the frontend, because it's public-facing and the less code the better. See [auth.lua](./frontend/auth.lua).

## Backend

<b>Golang</b> with the following packages:

- github.com/go-redis/redis
- golang.zx2c4.com/wireguard/wgctrl

If the backend receives a request, it generates the WireGuard keys, adds them to Redis, and updates the WireGuard interface. If it's an existing user, data is simply returned. Key rotation is taken care of using Redis `EXPIRE`.

The meat of backend is the [handleClient()](./backend/src/redis.go#L76) function. We can get the interface (and with it the server) for this client from its group. Then we pass the email (straight from the header of the request) and the server struct to this function, which decides what to do. For the email, note that we blindly accept whatever is coming from the OIDC provider.

# TODO

- Tests
- Email notifications
- Cleaning up
- Documentation
- Auto-firewall from `AllowedIPs`
