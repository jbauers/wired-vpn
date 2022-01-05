FROM golang as builder-mq
WORKDIR /tmp/mq

# Upstream directly would be nice, but it seems to be under heavy
# development, so keep it safe for now...
RUN git clone https://github.com/jbauers/message-queue .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o mq *.go

##########

FROM golang as builder-wired
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

COPY --from=builder-wired /tmp/backend/backend /opt/backend
COPY --from=builder-mq /tmp/mq/mq /opt/mq
