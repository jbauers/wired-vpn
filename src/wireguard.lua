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
local iputils  = require "resty.iputils"
local template = require "resty.template"

-- Expire the "ip" key in Redis after "key_ttl" seconds for automated key
-- rotation.
--
-- TODO: Do the same for the "uid" key, but keep the IP mappings longer.
-- FIXME: 30 seconds may be a bit too short for prod. A month is 2592000.
local key_ttl = 30

local view = template.new("wireguard.html", "layout.html")

-- Capture command output when running wg commands.
function run_cmd (cmd)
    local f = assert (io.popen (cmd))
    local s = f:read('*all')
    local res = string.gsub(s, "[\n\r]", "")
    f:close()
    return res
end

-- Our user is in the header, Nginx has done the auth.
view.uid = ngx.req.get_headers()['Authenticated-User']

-- Init our Redis connection
local red = redis:new()
red:set_timeout(1000)
local ok, err = red:connect("redis", 6379)
if not ok then
    view.err = "failed to connect: "..err
end

local res, err = red:publish("clients", view.uid)
if not res then
    view.err = "failed to publish "..err
end

local res, err = red:subscribe(view.uid)
if not res then
    view.err = "failed to subscribe "..err
end

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

local res, err = red:unsubscribe(view.uid)
if not res then
    view.err = "failed to subscribe "..err
end

-- Get data from Redis.
local res, err = red:get(view.uid)
if not res then
    view.err = "failed to get " ..view.uid..": "..err
end
view.ip = res

local res, err = red:hmget(view.ip, "privkey", "pubkey", "psk")
if not res then
    view.err = "failed to HMGet: "..err
end
view.privkey = res[1]
view.pubkey  = res[2]
view.psk     = res[3]

-- Close the Redis connection.
redis:close()

-- Render the page.
if not view.err then
    view.access = true
end
view:render()
