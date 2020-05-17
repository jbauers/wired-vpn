#!/bin/sh
source /opt/oidc.env

exec /usr/local/openresty/bin/openresty -g 'daemon off;'
