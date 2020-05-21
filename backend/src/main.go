package main

import (
	"net"
	"os"
	"time"
)

var serverInterface = os.Getenv("WG_INTERFACE")
var serverCIDR = os.Getenv("WG_NETWORK")
var serverIP = os.Getenv("WG_SERVER_IP")
var serverPort = 51820
var allowedIPs []net.IPNet

// Expiry of Redis keys for WireGuard key rotation.
var keyTTL = time.Duration(10 * time.Second)

// Maybe don't panic.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	privkey, pubkey := initServer()

	rc := redisClient()
	pubsub := rc.Subscribe("clients")
	pubsub.Receive()

	ch := pubsub.Channel()
	for msg := range ch {
		_ = handleClient(msg.Payload)
		rc.Publish(msg.Payload, pubkey)

		peerList := getPeerList()
		updateInterface(serverInterface, serverPort, privkey, peerList)
	}
}
