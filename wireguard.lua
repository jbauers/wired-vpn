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
-- FIXME: Generate IP
view.uid = ngx.req.get_headers()['Authenticated-User']
view.ip  = "10.0.0.2"

-- Init our Redis connection
local red = redis:new()
local ok, err = red:connect("redis", 6379)
if not ok then
    view.err = "failed to connect: "..err
end

-- Ensure the user exists in Redis.
local res, err = red:get(view.uid)
if not res then
    view.err = "failed to get " ..view.uid..": "..err
end

if res == ngx.null then
    ok, err = red:set(view.uid, view.ip)
    if not ok then
        view.err = "failed to set " ..view.uid..": "..err
    end
end

-- Serve/add user data from/to Redis.
local res, err = red:hmget(view.ip, "privkey", "pubkey", "psk")
if not res then
    view.err = "failed to HMGet: "..err
end

if res[1] == ngx.null then
    view.privkey = run_cmd("wg genkey")
    view.pubkey  = run_cmd("printf '"..view.privkey.."' | wg pubkey")
    view.psk     = run_cmd("wg genpsk")

    ok, err = red:hmset(view.ip, "privkey", view.privkey, "pubkey", view.pubkey, "psk", view.psk)
    if not ok then
        view.err = "failed to HMSet: "..err
    end
    ok, err = red:expire(view.ip, key_ttl)
    if not ok then
        view.err = "failed to set expire: "..err
    end
else
    view.privkey = res[1]
    view.pubkey  = res[2]
    view.psk     = res[3]
end

-- Close the Redis connection.
redis:set_keepalive(10000, 128)

-- Render the page.
if not view.err then
    view.access = true
end
view:render()
