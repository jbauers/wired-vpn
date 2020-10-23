local opts = {
    discovery     = os.getenv('OIDC_DISCOVERY_URL'),
    client_id     = os.getenv('OIDC_CLIENT_ID'),
    client_secret = os.getenv('OIDC_CLIENT_SECRET'),

    redirect_uri = ngx.var.scheme.."://"..ngx.var.server_name.."/redirect_uri",
    token_signing_alg_values_expected = "RS256",
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

ngx.req.set_header('Authenticated-User', res.user.email)
