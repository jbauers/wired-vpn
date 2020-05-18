package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/go-redis/redis"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// TODO: Env vars? Assign serverIP automatically.
var serverInterface = "wg0"
var serverCIDR = "10.100.0.0/24"
var serverIP = "10.100.0.1"

// Expiry of Redis keys for WireGuard key rotation.
var keyTTL = time.Duration(10 * time.Second)

// FIXME: Don't panic.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network address and broadcast address.
	lenIPs := len(ips)
	switch {
	case lenIPs < 2:
		return ips, nil

	default:
	return ips[1 : len(ips)-1], nil
	}
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func genKeys() (privkey string, pubkey string, psk string) {
	k, err := wgtypes.GeneratePrivateKey()
	check(err)

	sk, err := wgtypes.GenerateKey()
	check(err)

	privkey = k.String()
	pubkey = k.PublicKey().String()

	psk = sk.String()

	return privkey, pubkey, psk
}

func redisClient() (client *redis.Client) {
	client = redis.NewClient(&redis.Options{
		Addr:       "redis:6379",
		Password:   "",
		DB:         0,
		MaxRetries: 3,
	})
	return client
}

func handleClient(uid string) (ip string) {
	rc := redisClient()
	val, err := rc.Get(uid).Result()

	if err == redis.Nil || val == "" {
		ip, err = rc.SRandMember("availableIPs").Result()
		check(err)

		err = rc.SRem("availableIPs", ip).Err()
		check(err)

		err = rc.Set(uid, ip, 0).Err()
		check(err)
	} else {
		ip = val
	}

	if !handleClientKeys(ip) {
	        log.Print("NOTIFY: Already exists " + ip)
	} else {
	        log.Print("NOTIFY: Added " + ip)
	}

	return ip
}

func handleClientKeys(ip string) (bool) {
	rc := redisClient()
	res, err := rc.HMGet(ip, "pubkey", "privkey", "psk").Result()
	check(err)

	for _, r := range res {
		if r == nil {
			privkey, pubkey, psk := genKeys()
			_, err := rc.HMSet(ip, "pubkey", pubkey, "privkey", privkey, "psk", psk).Result()
			check(err)
			_, err = rc.Expire(ip, keyTTL).Result()
			check(err)

			return true
		}
	}

	return false
}


// FIXME: WIP.
func initServer() (pubkey string) {
	rc := redisClient()

	// Add all our IPs to this Redis key. We simply remove an entry
	// when adding new clients.
	availableIPs, err := hosts(serverCIDR)

	err = rc.SAdd("availableIPs", availableIPs).Err()
	check(err)

	err = rc.SRem("availableIPs", serverIP).Err()
	check(err)

	// Add the server data to Redis. Server keys don't expire automatically.
	_, err = rc.Get(serverIP).Result()
	if err == redis.Nil {
		err := rc.Set(serverInterface, serverIP, 0).Err()
		check(err)
	}

	privkey := ""

	res, err := rc.HMGet(serverIP, "pubkey", "privkey").Result()
	check(err)

	for _, r := range res {
		if r == nil {
			k, err := wgtypes.GeneratePrivateKey()
			check(err)
			privkey = k.String()
			pubkey = k.PublicKey().String()

			_, err = rc.HMSet(serverIP, "pubkey", pubkey, "privkey", privkey).Result()
			check(err)
		}
	}

	fmt.Println("------Backend ready-------")
	fmt.Println("PublicKey  = ", pubkey)
	fmt.Println("--------------------------")

	//
	// WIP!!!
	//

	// TODO:
	//  - Bring up the interface in an entrypoint, assigning keys
	//  - Continuously update peers
	//  - AllowedIPs
	//  - ...
	//

	wc, err := wgctrl.New()
	check(err)

	devices, err := wc.Devices()
	check(err)
	fmt.Println(devices)
	for _, d := range devices {
		fmt.Println(d.Name)
		fmt.Println(d.Type)
		fmt.Println(d.PrivateKey)
		fmt.Println(d.PublicKey)
		fmt.Println(d.ListenPort)
		fmt.Println(d.FirewallMark)
		fmt.Println(d.Peers)
	}

	return pubkey
}

func main() {
	serverkey := initServer()

	rc := redisClient()
	pubsub := rc.Subscribe("clients")
	pubsub.Receive()

	ch := pubsub.Channel()
	for msg := range ch {
		ip := handleClient(msg.Payload)
		fmt.Println(msg.Channel, msg.Payload, ip)
		rc.Publish(msg.Payload, serverkey)
	}
}
