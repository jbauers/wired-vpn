package main

import (
	"fmt"
	"net"
	"time"

	"github.com/go-redis/redis"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Expiry of Redis keys for WireGuard key rotation.
var keyTTL = time.Duration(10*time.Second)

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func nextIp(ip string) (nextIp string, valid bool) {
	valid = false
        res, net, err := net.ParseCIDR(ip)
	check(err)

	i := iterateIp(res)
	if net.Contains(i) {
		valid = true
	}

	nextIp = i.String()
	return nextIp, valid
}

func iterateIp(ip net.IP) net.IP {
	i := ip.To4()
	v := uint(i[0])<<24 + uint(i[1])<<16 + uint(i[2])<<8 + uint(i[3])
	v += 1
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	for (v3 == 0 || v3 == 255) {
		v3++
	}
	i = net.IPv4(v0, v1, v2, v3)
	return i
}

func genKeys() (privkey string, pubkey string, psk string) {
	k, err := wgtypes.GeneratePrivateKey()
	check(err)

	sk, err := wgtypes.GenerateKey()
	check(err)

	privkey = k.String()
	pubkey  = k.PublicKey().String()

	psk = sk.String()

	return privkey, pubkey, psk
}

func redisClient() (client *redis.Client) {
	client = redis.NewClient(&redis.Options{
		Addr:       "localhost:6379", // FIXME
		Password:   "",
		DB:         0,
		MaxRetries: 3,
	})
	return client
}

// FIXME: Ugly.
func handleClient (uid string) (string) {
	ip := "10.0.0.4/24"
	valid := false

	rc := redisClient()

	// TODO: Handle multiple servers.
	val, err := rc.Get(uid).Result()
	if err == redis.Nil {
		// TODO: Error handling.
		ip, valid = nextIp(ip)
		if !valid {
			return ""
		}
		err := rc.Set(uid, ip, 0).Err()
		check(err)
		_, err = rc.Expire(uid, keyTTL).Result()
		check(err)
	} else if err != nil {
		panic(err)
	} else {
		ip = val
	}

	return ip
}

func handleClientKeys (ip string) (present bool) {
	present = false

	rc := redisClient()

	res, err := rc.HMGet(ip, "pubkey", "privkey", "psk").Result()
	check(err)

	for _, r := range res {
	        if r == nil {
	        } else {
			present = true
		}
	}

	return present
}

// FIXME: WIP.
func printWireguardInfo() {

	// Generate server keys.
	k, err := wgtypes.GeneratePrivateKey()
	check(err)
	privkey := k.String()
	pubkey  := k.PublicKey().String()

	fmt.Println("------Server keys------")
	fmt.Println("PrivateKey = ", privkey)
	fmt.Println("PublicKey  = ", pubkey)
	fmt.Println("-----------------------")

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

	device, err := wc.Device("wg0")
	check(err)
	fmt.Println(device)
}

func main() {
	printWireguardInfo()

	rc := redisClient()
	pubsub := rc.Subscribe("clients")
	pubsub.Receive()

	ch := pubsub.Channel()
	for msg := range ch {
		ip := handleClient(msg.Payload)
		fmt.Println(msg.Channel, msg.Payload, ip)
		if !handleClientKeys(ip) {
			privkey, pubkey, psk := genKeys()
			_, err := rc.HMSet(ip, "pubkey", pubkey, "privkey", privkey, "psk", psk).Result()
			check(err)
			_, err = rc.Expire(ip, keyTTL).Result()
			check(err)
		}
		rc.Publish(msg.Payload, "ok")
	}
}

