-- Redis overview
-- --------------
--
-- uid -> ip, privkey, pubkey, psk
--
--        wg0              : 10.100.0.1/24 PRIVKEY, PUBKEY, PSK
--        test0@example.com: 10.100.0.2/32 PRIVKEY, PUBKEY, PSK
--
local cjson    = require "cjson"
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

-- Lower timeout for the pubsub reply.
-- FIXME: When the server is busy, these data may be skipped as
-- the reply is too late. The HMGet still works, but the server
-- metadata may be missing. Starts at ~500 clients (localhost).
red:set_timeout(100)
for i = 1, 2 do
    local res, err = red:read_reply()
    if not res then
        if err ~= "timeout" then
            view.err = "failed to read reply: "..err
            return
        end
    else
        data = cjson.decode(res[3])
        view.serverKey = data["Pubkey"]
        view.serverEndpoint = data["Endpoint"]
        view.serverPort = data["Port"]
    end
end
red:set_timeout(1000)

-- Once we got the "ok", we know the backend has added keys
-- for us, and don't care about our own channel anymore.
local res, err = red:unsubscribe(view.uid)
if not res then
    view.err = "failed to subscribe "..err
end

-- We can now fetch the data from Redis.
local res, err = red:hmget(view.uid, "ip", "privkey", "pubkey", "psk")
if not res then
    view.err = "failed to HMGet: "..err
end
view.ip      = res[1]
view.privkey = res[2]
view.pubkey  = res[3]
view.psk     = res[4]

-- Then we can terminate the Redis connection and render the page.
redis:close()
if not view.err then
    view.access = true
end
view:render()
