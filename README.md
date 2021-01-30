# Wired VPN

Docker based WireGuard server with OIDC support (Google SSO, OneLogin have been tested so far). A small Openresty frontend, a small Golang backend and Redis holding keys.

## Running

`./run.sh` (throwaway) or `docker-compose up` (persistent data).

### Frontend

<b>Openresty</b> with the following modules:

- lua-resty-http
- lua-resty-session
- lua-resty-openidc

Users are first authenticated with the OpenIDC module. There's some light validation of the OIDC provider's response before the user and its group are proxy passed as headers to the backend.

### Backend

Small <b>Golang</b> program with the following packages:

- github.com/go-redis/redis
- golang.zx2c4.com/wireguard/wgctrl

If the backend receives a request, it generates the WireGuard keys, adds them to Redis, and updates the WireGuard interface. If it's an existing user, data is simply returned. Key rotation is taken care of using Redis `EXPIRE`.

# Considerations

- The primary goal is simplicity
- Redis holds private keys of servers and clients

# TODO

- Cleaning up, documentation
- Firewall
