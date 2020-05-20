local opts = {
    discovery     = "https://accounts.google.com/.well-known/openid-configuration",
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

if not res.user then
    ngx.status = 400
    ngx.exit(ngx.HTTP_BAD_REQUEST)
end

ngx.req.set_header('Authenticated-User', res.user.email)
