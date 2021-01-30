-- Read settings.json.
local file = io.open("/settings.json", "rb")
if not file then return nil end
local settings = file:read "*a"
file:close()

-- Get OIDC options from settings.
local cjson = require "cjson"
local s = cjson.decode(settings)
local opts = {
    discovery     = s['oidc']['discovery_url'],
    client_id     = s['oidc']['client_id'],
    client_secret = s['oidc']['client_secret'],
    scope = "openid email profile groups",

    redirect_uri = ngx.var.scheme.."://"..ngx.var.server_name.."/redirect_uri",
    token_signing_alg_values_expected = "RS256",
}

-- Get valid groups from settings.
local groups = {}
for _, i  in pairs(s['interfaces']) do
    for _, g in pairs(i['groups']) do
        groups[g] = true
    end
end

-- Authenticate.
local res, err = require("resty.openidc").authenticate(opts)
if err or not res then
    ngx.status = 403
    ngx.exit(ngx.HTTP_FORBIDDEN)
end

if not res.user.email then
    ngx.status = 400
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

-- Validate.
local lastAt = res.user.email:find("[^%@]+$")
local domainPart = res.user.email:sub(lastAt, #res.user.email)

if domainPart ~= s['oidc']['allowed_email_domain'] then
    ngx.status = 403
    ngx.exit(ngx.HTTP_FORBIDDEN)
end

for _, group in pairs(res.user.groups) do
    if groups[group] then
        -- Set headers and proxy pass to our backend.
        ngx.req.set_header('X-Wired-User', res.user.email)
        ngx.req.set_header('X-Wired-Group', group)
        ngx.log(ngx.ALERT, 'Access granted: '..res.user.email..' - '..group)
        return
    end
end
