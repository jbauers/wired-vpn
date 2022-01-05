#!/bin/sh
HTTP_ENDPOINT="$(jq -r '.oidc.http_endpoint' /settings.json)"
sed -i "s/HTTP_ENDPOINT/${HTTP_ENDPOINT}/g" /usr/local/openresty/nginx/conf/nginx.conf

exec /usr/local/openresty/bin/openresty -g 'daemon off;'
