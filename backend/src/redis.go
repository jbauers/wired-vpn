package main

import (
	"encoding/base64"
	"log"
	"strings"

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

func initServer(rc *redis.Client) (privkey string, pubkey string) {
	err := rc.SAdd("usedIPs", serverIP).Err()
	check(err)

	res, err := rc.HMGet(serverInterface, "ip", "pubkey", "privkey").Result()
	check(err)

	if res[0] == nil {
		privkey, pubkey, _ = genKeys()
		data := map[string]interface{}{
			"ip": serverIP,
			"pubkey": pubkey,
			"privkey": privkey,
		}
		_, err = rc.HMSet(serverInterface, data).Result()
		check(err)
	}

	log.Print("-----------------------Backend ready------------------------")
	log.Print("  Interface  = " + serverInterface)
	log.Print("  Private IP = " + serverIP)
	log.Print("  PublicKey  = " + pubkey)
	log.Print("------------------------------------------------------------")

	return privkey, pubkey
}

func assignIP(rc *redis.Client) (ip string) {
	res, err := rc.SMembers("usedIPs").Result()
	check(err)

	ip = serverIP
	for stringInSlice(ip, res) {
		ip = incrementIP(ip)
	}

	err = rc.SAdd("usedIPs", ip).Err()
	check(err)

	return ip
}

func handleClient(uid string, serverInterface string, serverPort int, serverPrivkey string, rc *redis.Client) (error) {
	user, err := rc.HMGet(uid, "ip", "pubkey", "privkey", "psk").Result()
	check(err)

	ttl, err := rc.TTL(uid).Result()
	check(err)

	if ttl.Seconds() < minTTL || user[0] == nil {
		var peerList []wgtypes.PeerConfig

		if user[0] != nil {
			old_ip := user[0].(string)
			old_pubkey := user[1].(string)
			old_psk := user[3].(string)

			ref := old_ip + " " + old_pubkey + " " + old_psk + " " + uid
			b64 := base64.StdEncoding.EncodeToString([]byte(ref))

			err = rc.SRem("users", b64).Err()
			check(err)

			err = rc.SRem("usedIPs", old_ip).Err()
			check(err)

			old_config := getPeerConfig(old_ip, old_pubkey, old_psk, true)
			peerList = append(peerList, old_config)

			log.Print("ROTATED: " + uid + " - " + old_ip + " - " + old_pubkey)
		}

		privkey, pubkey, psk := genKeys()
		ip := assignIP(rc)

		data := map[string]interface{}{
			"ip": ip,
			"pubkey": pubkey,
			"privkey": privkey,
			"psk": psk,
		}
		_, err = rc.HMSet(uid, data).Result()
		check(err)

		// Add ip, pubkey and psk as base64 encoded string to Redis, so
		// we can get all in one go when updating the interface.
		s := ip + " " + pubkey + " " + psk + " " + uid
		b64 := base64.StdEncoding.EncodeToString([]byte(s))

		err = rc.SAdd("users", b64).Err()
		check(err)

		err = rc.Expire(uid, keyTTL).Err()
		check(err)

		config := getPeerConfig(ip, pubkey, psk, false)
		peerList = append(peerList, config)
		updateInterface(serverInterface, serverPort, serverPrivkey, peerList)

		log.Print("ADDED: " + uid + " - " + ip + " - " + pubkey)
	} else {
		ip := user[0].(string)
		pubkey := user[1].(string)

		log.Print("EXISTS: " + uid + " - " + ip + " - " + pubkey)
	}
	return nil
}

func getPeerList(rc *redis.Client) (peerList []wgtypes.PeerConfig) {
	users, err := rc.SMembers("users").Result()
	check(err)

	keys, err := rc.Keys("*@*").Result()
	check(err)

	for _, b64 := range users {
		if b64 == "" {
			log.Print("No peers!")
		} else {
			decoded, _ := base64.StdEncoding.DecodeString(b64)
			s := strings.Split(string(decoded), " ")

			if stringInSlice(s[3], keys) {
				config := getPeerConfig(s[0], s[1], s[2], false)
				peerList = append(peerList, config)

				log.Print("KEEP: " + s[3] + " - " + s[0] + " - " + s[1])
			} else {
				config := getPeerConfig(s[0], s[1], s[2], true)
				peerList = append(peerList, config)

				err = rc.SRem("users", b64).Err()
				check(err)

				err = rc.SRem("usedIPs", s[0]).Err()
				check(err)

				log.Print("REMOVED: " + s[3] + " - " + s[0] + " - " + s[1])
			}
		}
	}

	return peerList
}
