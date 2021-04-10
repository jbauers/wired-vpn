FROM golang as builder

WORKDIR /tmp/backend

RUN go mod init backend \
 && go get github.com/go-redis/redis/v8 \
 && go get golang.zx2c4.com/wireguard/wgctrl

COPY . /tmp/backend

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o backend *.go

##########
FROM alpine
RUN apk add --no-cache bash curl jq

COPY entrypoint.sh /
ENTRYPOINT /entrypoint.sh

COPY --from=builder /tmp/backend/backend /opt/backend
