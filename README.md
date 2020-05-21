# Wired VPN

# Very much WIP!!

Definitely multiple dragons here.

# Overview

Docker based WireGuard server with OIDC support (Google SSO has been tested so far). A small Openresty frontend, a small Golang backend and Redis in between.

## Running

`./run.sh` (throwaway) or `docker-compose up` (persistent data).

### Frontend

<b>Openresty</b> with the following modules:

- lua-resty-http
- lua-resty-session
- lua-resty-openidc
- lua-resty-template

Users are authenticated with the OpenIDC module. See `nginx.conf`. Google SSO has been tested so far.

After authentication, Nginx passes our `Authenticated-User` to `src/wireguard.lua`. This script talks to our backend using Redis Pub/Sub. It fetches all information from Redis and adds the content for the page.

### Backend

Small <b>Golang</b> program with the following packages:

- github.com/go-redis/redis
- golang.zx2c4.com/wireguard/wgctrl

If the backend receives a Redis message on a specific channel, it checks whether a user exists. If not, it generates the WireGuard keys, adds them to Redis, and updates the WireGuard interface. It then pings Openresty that the work is done. Key rotation is taken care of using Redis `EXPIRE`.

# TODO

- Cleaning up, documentation
- Firewall rules
- DNS, AllowedIPs client-side
