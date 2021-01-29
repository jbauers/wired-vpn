#!/bin/sh
HTTP_SERVER_NAME="$(jq '.oidc.http_domain' /settings.json)"
sed -i "s/HTTP_SERVER_NAME/${HTTP_SERVER_NAME}/g" /usr/local/openresty/nginx/conf/nginx.conf

exec /usr/local/openresty/bin/openresty -g 'daemon off;'
