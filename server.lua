local redis    = require "resty.redis"
local iputils  = require "resty.iputils"

function run_cmd (cmd)
    local f = assert (io.popen (cmd))
    local s = f:read('*all')
    local res = string.gsub(s, "[\n\r]", "")
    f:close()
    return res
end

local uid  = "wg0"
local ip   = "10.200.0.1/24"
local port = "51280"

local privkey = run_cmd("wg genkey")
local pubkey  = run_cmd("printf '"..privkey.."' | wg pubkey")
local psk     = run_cmd("wg genpsk")

local file = '/tmp/'..uid..'.conf'
local config = assert (io.open(file, "a"))
io.output(config)
io.write("[Interface]")
io.write("PrivateKey = "..privkey)
io.write("Address = "..ip)
io.write("ListenPort = "..port)
io.close(config)

local red = redis:new()
local ok, err = red:connect("redis", 6379)
if not ok then
    ngx.say("failed to connect: ", err)
end

redis:close()
