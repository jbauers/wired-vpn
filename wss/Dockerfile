FROM golang as builder
WORKDIR /tmp/wss

RUN go mod init wss \
 && go get github.com/gorilla/websocket

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o wss *.go
