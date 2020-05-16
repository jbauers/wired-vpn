local redis    = require "resty.redis"
local iputils  = require "resty.iputils"
local template = require "resty.template"

local view = template.new("wireguard.html", "layout.html")
local uid = ngx.req.get_headers()['Authenticated-User']

-- Generate our PrivateKey
local f = assert (io.popen ("wg genkey"))
local privkey = f:read('*all')
f:close()

-- Generate our PublicKey
local f = assert (io.popen ("printf '"..privkey.."' | wg pubkey"))
local pubkey = f:read('*all')
f:close()

local ip = "10.0.0.2"

view.message = uid..' - '..ip..' - '..privkey..' - '..pubkey

-- Init our Redis connection
local red = redis:new()
red:set_timeouts(1000, 1000, 1000) -- 1 sec
local ok, err = red:connect("redis", 6379)
if not ok then
    view.message = "failed to connect: "..err
    return
end

-- Add our data to Redis
ok, err = red:hmset(uid, "ip", ip, "privkey", privkey, "pubkey", pubkey)
if not ok then
    view.message = "failed to HMSet: "..err
    return
end

-- Get our data from Redis and do a sanity check
local res, err = red:hmget(uid, "ip", "privkey", "pubkey")
if not res then
    view.message = "failed to HMGet: "..err
    return
end

if not (privkey == res[2]) or not (pubkey == res[3]) then
    view.message = "mismatch between server and client keys"
    return
end

-- Finally, render the page
view:render()
