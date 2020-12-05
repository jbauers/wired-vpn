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

func initServer(rc *redis.Client) (privateKey string, publicKey string) {
	err := rc.SAdd("usedIPs", serverIP).Err()
	check(err)

	res, err := rc.HMGet(serverInterface, "ip", "pubkey", "privkey").Result()
	check(err)

	if res[0] == nil {
		privateKey, publicKey, _ = genKeys()
		peer := map[string]interface{}{
			"ip":      serverIP,
			"pubkey":  publicKey,
			"privkey": privateKey,
		}
		err = rc.HMSet(serverInterface, peer).Err()
		check(err)
	}

	return privateKey, publicKey
}

func assignIP(rc *redis.Client) (ip string, err error) {
	ips, err := rc.SMembers("usedIPs").Result()
	check(err)

	ip, err = getAvailableIP(ips)
	if err != nil {
		return "", err
	}

	err = rc.SAdd("usedIPs", ip).Err()
	check(err)

	return ip, nil
}

func handleClient(uid string, server Peer) (err error, ip string, publicKey string, privateKey string, presharedKey string) {
	rc := server.RedisClient
	user, err := rc.HMGet(uid, "ip", "pubkey", "privkey", "psk").Result()
	check(err)

	ttl, err := rc.TTL(uid).Result()
	check(err)

	if ttl.Seconds() < minTTL || user[0] == nil {
		var peerList []wgtypes.PeerConfig

		if user[0] != nil {
			staleIP := user[0].(string)
			stalePublicKey := user[1].(string)
			stalePresharedKey := user[3].(string)

			ref := staleIP + " " + stalePublicKey + " " + stalePresharedKey + " " + uid
			b64 := base64.StdEncoding.EncodeToString([]byte(ref))

			// Remove stale base64 string from Redis.
			err = rc.SRem("users", b64).Err()
			check(err)

			// Free up IP.
			err = rc.SRem("usedIPs", staleIP).Err()
			check(err)

			// Add stale config to peerList with toRemove set to true. We'll also add the
			// new config to peerList, so when we update the interface at the end, we're
			// rotating the config for this peer.
			stalePeerConfig := getPeerConfig(staleIP, stalePublicKey, stalePresharedKey, true)
			peerList = append(peerList, stalePeerConfig)

			log.Printf("Rotating WireGuard peer %s %s %s", uid, staleIP, stalePublicKey)
		}

		privateKey, publicKey, presharedKey = genKeys()

		ip, err = assignIP(rc)
		if err != nil {
			return err, "", "", "", ""
		}

		peer := map[string]interface{}{
			"ip":      ip,
			"pubkey":  publicKey,
			"privkey": privateKey,
			"psk":     presharedKey,
		}
		err = rc.HMSet(uid, peer).Err()
		check(err)

		// Add ip, pubkey and psk as base64 encoded string to Redis, so
		// we can get all in one go when updating the interface.
		s := ip + " " + publicKey + " " + presharedKey + " " + uid
		b64 := base64.StdEncoding.EncodeToString([]byte(s))

		err = rc.SAdd("users", b64).Err()
		check(err)

		// Expire the uid key in Redis. When it is found missing when
		// getPeerList is called, the other Redis keys will be removed.
		err = rc.Expire(uid, keyTTL).Err()
		check(err)

		peerConfig := getPeerConfig(ip, publicKey, presharedKey, false)
		peerList = append(peerList, peerConfig)

		err = updateInterface(server, peerList)
		check(err)

		log.Printf("Added WireGuard peer %s %s %s", uid, ip, publicKey)
		log.Printf("Updated WireGuard interface %s", server.Interface)
	} else {
		ip = user[0].(string)
		publicKey = user[1].(string)
		privateKey = user[2].(string)
		presharedKey = user[3].(string)

		log.Printf("Found existing WireGuard peer %s %s %s", uid, ip, publicKey)
	}
	return nil, ip, publicKey, privateKey, presharedKey
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

			// When both the base64 string and uid key exist, we'll keep the
			// config. When the uid key has expired, we'll remove the stale entries.
			if stringInSlice(s[3], keys) {
				peerConfig := getPeerConfig(s[0], s[1], s[2], false)
				peerList = append(peerList, peerConfig)

				log.Printf("Keeping WireGuard peer %s %s %s", s[3], s[0], s[1])
			} else {
				peerConfig := getPeerConfig(s[0], s[1], s[2], true)
				peerList = append(peerList, peerConfig)

				err = rc.SRem("users", b64).Err()
				check(err)

				err = rc.SRem("usedIPs", s[0]).Err()
				check(err)

				log.Printf("Removing WireGuard peer %s %s %s", s[3], s[0], s[1])
			}
		}
	}

	return peerList
}
