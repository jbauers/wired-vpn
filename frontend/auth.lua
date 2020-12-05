local opts = {
    discovery     = os.getenv('OIDC_DISCOVERY_URL'),
    client_id     = os.getenv('OIDC_CLIENT_ID'),
    client_secret = os.getenv('OIDC_CLIENT_SECRET'),
    scope = "openid email profile groups",

    redirect_uri = ngx.var.scheme.."://"..ngx.var.server_name.."/redirect_uri",
    token_signing_alg_values_expected = "RS256",
}

-- Map OIDC groups to WireGuard interfaces. The backend will manage the peers
-- for each interface, and we can then apply different firewall rules per
-- interface using Shorewall.
-- FIXME:
--   - Implement multiple interfaces in the backend
--   - Add working Shorewall firewall configs
--   - Tie things together with a "high-level" config
local groups = {
    Engineering = "wg0",
}

local res, err = require("resty.openidc").authenticate(opts)

if err or not res then
    ngx.status = 403
    ngx.exit(ngx.HTTP_FORBIDDEN)
end

if not res.user.email then
    ngx.status = 400
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

local lastAt = res.user.email:find("[^%@]+$")
local domainPart = res.user.email:sub(lastAt, #res.user.email)

if domainPart ~= os.getenv('EMAIL_DOMAIN') then
    ngx.status = 403
    ngx.exit(ngx.HTTP_FORBIDDEN)
end

ngx.req.set_header('X-Wired-User', res.user.email)

for index, group in pairs(res.user.groups) do
    if not groups[group] then
        ngx.log(ngx.ALERT, 'Access denied for '..res.user.email..' - '..group)
    else
        ngx.log(ngx.ALERT, 'Access granted for '..res.user.email..' - '..group..' - '..groups[group])
        ngx.req.set_header('X-Wired-Interface', groups[group])
        return
    end
end
