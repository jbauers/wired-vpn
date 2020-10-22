package main

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

var serverInterface = os.Getenv("WG_SERVER_INTERFACE")
var serverCIDR = os.Getenv("WG_SERVER_CIDR")
var serverIP = os.Getenv("WG_SERVER_IP")
var serverEndpoint = os.Getenv("WG_SERVER_ENDPOINT")

// Expiry of Redis keys for WireGuard key rotation. We expire the "uid"
// key after the keyTTL value. Upon interface update, when the "uid"
// is missing, but present as part of the "users" SMEMBERS, we will
// free up the IP from "usedIPs" and remove the stale config.
var keyTTL = time.Duration(20 * time.Second)

// If a request comes in and the TTL for its "uid" key is less than this
// minTTL value, the WireGuard keys will be rotated. If no request comes
// in until the key is expired, it will be removed (as described above).
var minTTL = float64(10)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	type serverInfo struct {
		Pubkey   string
		Endpoint string
		Port     int
	}

	serverPort, err := strconv.Atoi(os.Getenv("WG_SERVER_PORT"))
	check(err)

	rc := redisClient()
	serverPrivkey, serverPubkey := initServer(rc)

	info := serverInfo{
		Pubkey:   serverPubkey,
		Endpoint: serverEndpoint,
		Port:     serverPort,
	}

	jsonData, err := json.Marshal(info)
	check(err)

	go func() {
		for true {
			time.Sleep(10 * time.Second)
			peerList := getPeerList(rc)
			updateInterface(serverInterface, serverPort, serverPrivkey, peerList)
		}
	}()

	pubsub := rc.Subscribe("clients")
	pubsub.Receive()

	ch := pubsub.Channel()
	for msg := range ch {
		err := handleClient(msg.Payload, rc)
		check(err)

		err = rc.Publish(msg.Payload, jsonData).Err()
		check(err)
	}
}
