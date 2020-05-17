# Wired VPN

## <b>WIP</b> note!!

`./run.sh` to start the frontend

`./test.sh` to start the backend

# Overview

### Frontend

<b>Openresty</b> with the following modules:

- lua-resty-http
- lua-resty-session
- lua-resty-openidc
- lua-resty-template

Users are authenticated with the OpenIDC module. See `nginx.conf`. Google SSO has been tested so far.

After authentication, Nginx passes our `Authenticated-User` to `src/wireguard.lua`. This script talks to our backend using Redis Pub/Sub. It fetches all information from Redis and adds the content for the page.

### Backend

The backend is a small Go program. If it receives a message on a specific Redis channel, it checks whether a user exists in Redis. If not, it generates the WireGuard keys and adds them to Redis. It then pings Openresty that the work is done. Key rotation is taken care of using Redis `EXPIRE`.

# TODO

No interface is currently configured, no peers are added. This is the basic Pub/Sub for the frontend and backend.

- Bringing up the WireGuard server
- Adding firewall rules
- Updating server configurations on changes
- Lots more...
