FROM golang as builder
WORKDIR /tmp/vpn

RUN go mod init vpn \
 && go get github.com/gorilla/websocket \
 && go get golang.zx2c4.com/wireguard/wgctrl

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o vpn *.go

##########

FROM alpine
RUN apk add --no-cache bash wireguard-tools

COPY entrypoint.sh /
ENTRYPOINT /entrypoint.sh

COPY --from=builder /tmp/vpn/vpn /opt/vpn
