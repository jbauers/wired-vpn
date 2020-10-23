FROM golang as builder

WORKDIR /tmp/backend

RUN go mod init backend \
 && go get github.com/go-redis/redis/v8 \
 && go get golang.zx2c4.com/wireguard/wgctrl

COPY src /tmp/backend

RUN CGO_ENABLED=0 GOOS=linux \
    go build *.go

##########
FROM alpine
RUN apk add --no-cache bash shorewall

COPY entrypoint.sh /
ENTRYPOINT /entrypoint.sh

COPY rootfs/ /
COPY --from=builder /tmp/backend/main /opt/backend
