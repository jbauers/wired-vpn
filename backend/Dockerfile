FROM golang as builder

RUN go get github.com/go-redis/redis
RUN go get golang.zx2c4.com/wireguard/wgctrl

COPY backend.go .
RUN CGO_ENABLED=0 GOOS=linux \
    go build backend.go

FROM alpine
COPY --from=builder /go/backend /opt/
CMD /opt/backend
