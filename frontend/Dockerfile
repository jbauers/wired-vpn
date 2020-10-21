FROM openresty/openresty:alpine-fat

RUN apk add --no-cache \
  ca-certificates

RUN luarocks install lua-resty-http \
 && luarocks install lua-resty-session \
 && luarocks install lua-resty-openidc \
 && luarocks install lua-resty-template

COPY nginx.conf /usr/local/openresty/nginx/conf/

COPY templates /opt/templates
COPY css /opt/css
COPY src/ /opt/
