FROM golang as builder

RUN go get github.com/go-redis/redis
RUN go get golang.zx2c4.com/wireguard/wgctrl

COPY src .
RUN CGO_ENABLED=0 GOOS=linux \
    go build *.go

##########
FROM alpine
RUN apk add --no-cache bash

COPY entrypoint.sh /
ENTRYPOINT /entrypoint.sh

COPY --from=builder /go/main /opt/backend
