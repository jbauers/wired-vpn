package main

import (
	"log"

	"github.com/go-redis/redis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func redisClient() (client *redis.Client) {
	client = redis.NewClient(&redis.Options{
		Addr:       "redis:6379",
		Password:   "",
		DB:         0,
		MaxRetries: 3,
	})
	return client
}

func handleClient(uid string) (ip string, config wgtypes.PeerConfig) {
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

	added, config := handleClientKeys(ip)
	if !added {
		log.Print(uid + " already exists: " + ip)
	} else {
		log.Print(uid + " added: " + ip)
	}

	log.Print(config)
	return ip, config
}

func handleClientKeys(ip string) (added bool, config wgtypes.PeerConfig) {
	added = false
	rc := redisClient()
	res, err := rc.HMGet(ip, "pubkey", "privkey", "psk").Result()
	check(err)

	privkey := ""
	pubkey := ""
	psk := ""
	for p, r := range res {
		if r == nil {
			privkey, pubkey, psk = genKeys()
			_, err := rc.HMSet(ip, "pubkey", pubkey, "privkey", privkey, "psk", psk).Result()
			check(err)
			_, err = rc.Expire(ip, keyTTL).Result()
			check(err)

			added = true
		} else {
			// See HMGet above for indexes
			if p == 0 {
				pubkey = r.(string)
			}
			if p == 2 {
				psk = r.(string)
			}
		}
	}
	config = generatePeerConfig(pubkey, psk)
	return added, config
}

func initServer() (privkey string, pubkey string) {
	rc := redisClient()
	availableIPs, err := hosts(serverCIDR)

	err = rc.SAdd("availableIPs", availableIPs).Err()
	check(err)
	err = rc.SRem("availableIPs", serverIP).Err()
	check(err)

	_, err = rc.Get(serverIP).Result()
	if err == redis.Nil {
		err := rc.Set(serverInterface, serverIP, 0).Err()
		check(err)
	}

	res, err := rc.HMGet(serverIP, "pubkey", "privkey").Result()
	check(err)

	for _, r := range res {
		if r == nil {
			privkey, pubkey, _ = genKeys()
			_, err = rc.HMSet(serverIP, "pubkey", pubkey, "privkey", privkey).Result()
			check(err)
		}
	}

	log.Print("-----------------------Backend ready------------------------")
	log.Print("  PublicKey = " + pubkey)
	log.Print("------------------------------------------------------------")

	return privkey, pubkey
}
