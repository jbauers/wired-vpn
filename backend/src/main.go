package main

import (
	"encoding/json"
	"os"
	"time"
	"strconv"
)

var serverInterface = os.Getenv("WG_SERVER_INTERFACE")
var serverCIDR = os.Getenv("WG_SERVER_CIDR")
var serverIP = os.Getenv("WG_SERVER_IP")
var serverEndpoint = os.Getenv("WG_SERVER_ENDPOINT")

// Expiry of Redis keys for WireGuard key rotation.
var keyTTL = time.Duration(10 * time.Second)

// Maybe don't panic.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	serverPrivkey, serverPubkey := initServer()

	serverPort, err := strconv.Atoi(os.Getenv("WG_SERVER_PORT"))
	check(err)

	rc := redisClient()
	pubsub := rc.Subscribe("clients")
	pubsub.Receive()

	type serverInfo struct {
		Pubkey string
		Endpoint string
		Port int
	}

	info := serverInfo{
		Pubkey: serverPubkey,
		Endpoint: serverEndpoint,
		Port: serverPort,
	}

	jsonData, err := json.Marshal(info)

	ch := pubsub.Channel()
	for msg := range ch {
		_ = handleClient(msg.Payload)
		err := rc.Publish(msg.Payload, jsonData).Err()
		check(err)

		peerList := getPeerList()
		updateInterface(serverInterface, serverPort, serverPrivkey, peerList)
	}
}
