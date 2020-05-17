-- Redis overview
-- --------------
--
-- uid -> ip
--
--        wg1:               10.100.0.1/24
--        email@example.com: 10.100.0.2/32
--
-- ip  -> privkey, pubkey, endpoint, port, allowed_ips
--
--        10.100.0.1/24: PRIVKEY, PUBKEY, 192.168.100.1, 51820, 10.0.0.0/16
--        10.100.0.2/32: PRIVKEY, PUBKEY, 192.168.100.1, 51820, 10.0.0.0/16
--

local redis    = require "resty.redis"
local template = require "resty.template"

local view = template.new("wireguard.html", "layout.html")

-- Nginx has done the auth and added the email address to this
-- header.
view.uid = ngx.req.get_headers()['Authenticated-User']

-- Init our Redis Pub/Sub.
local red = redis:new()
red:set_timeout(1000)
local ok, err = red:connect("redis", 6379)
if not ok then
    view.err = "failed to connect: "..err
end

-- Send our email address to the "clients" channel, where our
-- backend is listening for messages.
local res, err = red:publish("clients", view.uid)
if not res then
    view.err = "failed to publish "..err
end

-- Once we've announces us as a client, subscribe to a channel
-- with our email address as the key. Our backend replies with
-- an "ok" on this channel.
local res, err = red:subscribe(view.uid)
if not res then
    view.err = "failed to subscribe "..err
end

-- This is needed to render the page AFTER we got a reply from
-- the server. No idea why it has to be like this specifically,
-- but it works. TODO: Someone explain and optimize if needed.
red:set_timeout(10)
for i = 1, 2 do
    local res, err = red:read_reply()
    if not res then
        if err ~= "timeout" then
            view.err = "failed to read reply: "..err
            return
        end
    end
end
red:set_timeout(1000)

-- Once we got the "ok", we know the backend has added keys
-- for us, and don't care about our own channel anymore.
local res, err = red:unsubscribe(view.uid)
if not res then
    view.err = "failed to subscribe "..err
end

-- We can now fetch the data from Redis. We need our IP first,
-- which is the key to our hash map with our WireGuard keys.
local ip, err = red:get(view.uid)
if not ip then
    view.err = "failed to get " ..view.uid..": "..err
end
view.ip = ip

local res, err = red:hmget(ip, "privkey", "pubkey", "psk")
if not res then
    view.err = "failed to HMGet: "..err
end
view.privkey = res[1]
view.pubkey  = res[2]
view.psk     = res[3]

-- Then we can terminate the Redis connection and render the page.
redis:close()
if not view.err then
    view.access = true
end
view:render()
