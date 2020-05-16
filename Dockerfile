FROM golang as builder

RUN go get github.com/go-redis/redis
RUN go get golang.zx2c4.com/wireguard/wgctrl

COPY src/handler.go .
RUN CGO_ENABLED=0 GOOS=linux \
    go build handler.go

######

FROM openresty/openresty:alpine-fat

RUN apk add --no-cache \
  ca-certificates \
  shorewall \
  wireguard-tools

RUN luarocks install lua-resty-http \
 && luarocks install lua-resty-session \
 && luarocks install lua-resty-openidc \
 && luarocks install lua-resty-template \
 && luarocks install lua-resty-iputils

COPY entrypoint.sh /
ENTRYPOINT /entrypoint.sh

COPY nginx.conf /usr/local/openresty/nginx/conf/

COPY templates /opt/templates
COPY oidc.env src/wireguard.lua /opt/
COPY --from=builder /go/handler /opt/
