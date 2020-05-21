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

func handleClient(uid string) (ip string) {
	rc := redisClient()
	val, err := rc.Get(uid).Result()

	if err == redis.Nil || val == "" {
		// Get a random IP from the available IP pool.
		// FIXME: Performance on a /16 network. Don't even dare /8.
		ip, err = rc.SRandMember("availableIPs").Result()
		check(err)

		// Remove the assigned IP from the pool.
		err = rc.SRem("availableIPs", ip).Err()
		check(err)

		// Add them to the used IP pool.
		err = rc.SAdd("usedIPs", ip).Err()
		check(err)

		// Add the UID -> IP mapping.
		err = rc.Set(uid, ip, 0).Err()
		check(err)

		log.Print("Added user " + uid + " with IP " + ip)

	} else {
		ip = val
		log.Print("Existing user " + uid + " with IP " + ip)
	}

	return ip
}

func handleClientKeys(ip string) (config wgtypes.PeerConfig) {
	rc := redisClient()
	res, err := rc.HMGet(ip, "pubkey", "privkey", "psk").Result()
	check(err)

	privkey := ""
	pubkey := ""
	psk := ""

	for p, r := range res {
		if r == nil {
			// Generate all keys.
			privkey, pubkey, psk = genKeys()

			// Add the generated keys to Redis.
			_, err := rc.HMSet(ip, "pubkey", pubkey, "privkey", privkey, "psk", psk).Result()
			check(err)

			// Set an expiry for our keys, so Redis automatically cleans up.
			_, err = rc.Expire(ip, keyTTL).Result()
			check(err)
		} else {
			// See the HMGet above for indexes.
			if p == 0 {
				pubkey = r.(string)
			}
			if p == 2 {
				psk = r.(string)
			}
		}
	}

	config = generatePeerConfig(ip, pubkey, psk)
	return config
}

func getPeerList() (peerList []wgtypes.PeerConfig) {
	rc := redisClient()

	res, err := rc.SMembers("usedIPs").Result()
	check(err)

	for _, r := range res {
		if r == "" {
			log.Print("No peers!")
		} else {
			config := handleClientKeys(r)
			peerList = append(peerList, config)
		}
	}

	return peerList
}

func initServer() (privkey string, pubkey string) {
	rc := redisClient()

	// This adds all hosts in our subnet.
	// FIXME: RIP on a /8, unsurprisingly. Maybe it's better to
	// iterate 1 by 1 and keep track elsewhere.
	availableIPs, err := hosts(serverCIDR)
	err = rc.SAdd("availableIPs", availableIPs).Err()
	check(err)

	// Remove the server IP from the available IP pool.
	err = rc.SRem("availableIPs", serverIP).Err()
	check(err)

	// Add the server configuration to Redis.
	err = rc.Get(serverIP).Err()
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

	// We're ready!
	log.Print("-----------------------Backend ready------------------------")
	log.Print("  PublicKey = " + pubkey)
	log.Print("------------------------------------------------------------")

	return privkey, pubkey
}
