#FROM golang as builder
#
#RUN go get github.com/dgrijalva/jwt-go \
#           github.com/crewjam/httperr \
#           github.com/crewjam/saml
#
#COPY wg-tools/main.go .
#RUN CGO_ENABLED=0 GOOS=linux \
#    go build main.go
#

######

#WORKDIR /app
#COPY --from=builder /go/main .

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
COPY oidc.env wireguard.lua /opt/
