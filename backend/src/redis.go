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

func assignIP(rc *redis.Client) (ip string) {
	res, err := rc.SMembers("usedIPs").Result()
	check(err)

	for _, r := range res {
		ip = incrementIP(r)
		for stringInSlice(ip, res) {
			ip = incrementIP(ip)
		}
	}

	err = rc.SAdd("usedIPs", ip).Err()
	check(err)

	return ip
}

func handleClient(uid string, rc *redis.Client) (config wgtypes.PeerConfig) {
	user, err := rc.HMGet(uid, "ip", "pubkey", "privkey", "psk").Result()
	check(err)

	ip := ""
	pubkey := ""
	privkey := ""
	psk := ""

	if user[0] == nil {
		privkey, pubkey, psk = genKeys()
		ip = assignIP(rc)

		_, err = rc.HMSet(uid, "ip", ip, "pubkey", pubkey, "privkey", privkey, "psk", psk).Result()
		check(err)

		err = rc.SAdd("users", uid).Err()
		check(err)

		log.Print(" +++  ADD  --- " + uid + " - " + ip + " - " + pubkey)

		// Set an expiry for our keys, so Redis automatically cleans up.
		// FIXME: Clean up usedIPs.
		//
		// _, err = rc.Expire(uid, keyTTL).Result()
		// check(err)

	} else {
		for i, r := range user {
			if i == 0 {
				ip = r.(string)
			}
			if i == 1 {
				pubkey = r.(string)
			}
			if i == 3 {
				psk = r.(string)
			}
		}
		log.Print(" +++ EXIST --- " + uid + " - " + ip + " - " + pubkey)
	}

	config = getPeerConfig(ip, pubkey, psk)
	return config
}

func getPeerList(rc *redis.Client) (peerList []wgtypes.PeerConfig) {
	res, err := rc.SMembers("users").Result()
	check(err)

	for _, r := range res {
		if r == "" {
			log.Print("No peers!")
		} else {
			config := handleClient(r, rc)
			peerList = append(peerList, config)
		}
	}

	return peerList
}

func initServer(rc *redis.Client) (privkey string, pubkey string) {
	err := rc.SAdd("usedIPs", serverIP).Err()
	check(err)

	res, err := rc.HMGet(serverInterface, "ip", "pubkey", "privkey").Result()
	check(err)

	if res[0] == nil {
		privkey, pubkey, _ = genKeys()
		_, err = rc.HMSet(serverInterface, "ip", serverIP, "pubkey", pubkey, "privkey", privkey).Result()
		check(err)
	}

	log.Print("-----------------------Backend ready------------------------")
	log.Print("  PublicKey = " + pubkey)
	log.Print("------------------------------------------------------------")

	return privkey, pubkey
}
