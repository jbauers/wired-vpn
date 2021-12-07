FROM golang as builder
WORKDIR /tmp/wss

RUN go mod init wss \
 && go get github.com/gorilla/websocket \
 && go get golang.zx2c4.com/wireguard/wgctrl

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o wss *.go

##########

FROM alpine
RUN apk add --no-cache bash wireguard-tools

COPY entrypoint.sh /
ENTRYPOINT /entrypoint.sh

COPY --from=builder /tmp/wss/wss /opt/wss
