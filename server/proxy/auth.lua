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
    accept_unsupported_alg = false,
}

-- Get valid groups from settings.
local groups = {}
for _, g in pairs(s['oidc']['allowed_groups']) do
    groups[g] = true
end

-- Get the public key before authenticating.
local public_key = ngx.var.arg_public_key

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
local last_at = res.user.email:find("[^%@]+$")
local domain_part = res.user.email:sub(last_at, #res.user.email)

if domain_part ~= s['oidc']['allowed_email_domain'] then
    ngx.status = 403
    ngx.exit(ngx.HTTP_FORBIDDEN)
end

for _, group in pairs(res.user.groups) do
    if groups[group] then
        ngx.req.set_header('X-Wired-User', res.user.email)
        ngx.req.set_header('X-Wired-Group', group)
        ngx.req.set_header('X-Wired-Public-Key', public_key)
        ngx.log(ngx.ALERT, 'Access granted: '..res.user.email..' - '..group..' - '..public_key)
        return
    end
end

ngx.status = 403
ngx.exit(ngx.HTTP_FORBIDDEN)