FROM openresty/openresty:alpine-fat

RUN apk add --no-cache \
  ca-certificates

RUN luarocks install lua-resty-http \
 && luarocks install lua-resty-session \
 && luarocks install lua-resty-openidc

COPY entrypoint.sh /
ENTRYPOINT /entrypoint.sh

COPY nginx.conf /usr/local/openresty/nginx/conf/
COPY auth.lua css/ /opt/
