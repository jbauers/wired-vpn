package main

import (
	"encoding/base64"
	"log"
	"strings"

	"github.com/go-redis/redis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Returns a new Redis client.
func redisClient() (client *redis.Client) {
	client = redis.NewClient(&redis.Options{
		Addr:       "redis:6379",
		Password:   "",
		DB:         0,
		MaxRetries: 3,
	})
	return client
}

// Initialises the server, adding the server IP, public key and private key to
// Redis, and returning the keys as strings. FIXME: Try fetching existing data
// from Redis, too...
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

// Assigns a free IP to a peer, returning the IP and an error. The usedIPs key
// keeps the currently assigned IPs as a set in Redis. Assigning or freeing up
// an IP is a matter of modifying this set.
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

// Accepts the uid - an email - and "handles" this peer on the server. It will
// either just check Redis and simply return the data, or add a new peer config
// and update the server's interface. It also takes care of rotating configs
// that are expiring soon. In all cases, an error, the IP, and all keys for the
// peer are returned to be served by the web server.
func handleClient(uid string, server Peer) (err error, ip string, publicKey string, privateKey string, presharedKey string) {
	rc := server.RedisClient
	user, err := rc.HMGet(uid, "ip", "pubkey", "privkey", "psk").Result()
	check(err)

	ttl, err := rc.TTL(uid).Result()
	check(err)

	// Either a new user, or this user's config is expiring soon. In both
	// cases we need a new config.
	if ttl.Seconds() < minTTL || user[0] == nil {
		var peerList []wgtypes.PeerConfig

		// An existing user. Clean up the config that's expiring.
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

			// Add stale config to peerList with toRemove
			// set to true.
			stalePeerConfig := getPeerConfig(staleIP, stalePublicKey, stalePresharedKey, true)
			peerList = append(peerList, stalePeerConfig)

			log.Printf("Rotating WireGuard peer %s %s %s", uid, staleIP, stalePublicKey)
		}

		// Generate new keys and assign a free IP.
		privateKey, publicKey, presharedKey = genKeys()

		ip, err = assignIP(rc)
		if err != nil {
			return err, "", "", "", ""
		}

		// Add the uid key with our IP and keys to Redis.
		peer := map[string]interface{}{
			"ip":      ip,
			"pubkey":  publicKey,
			"privkey": privateKey,
			"psk":     presharedKey,
		}
		err = rc.HMSet(uid, peer).Err()
		check(err)

		// Expire the uid key after keyTTL. When it is found missing
		// when getPeerList is called, the other Redis keys will also
		// be removed.
		err = rc.Expire(uid, keyTTL).Err()
		check(err)

		// Add IP, public key and pre-shared key as a base64 encoded
		// string to Redis.
		s := ip + " " + publicKey + " " + presharedKey + " " + uid
		b64 := base64.StdEncoding.EncodeToString([]byte(s))

		err = rc.SAdd("users", b64).Err()
		check(err)

		// Add the new config to peerList with toRemove set to false.
		peerConfig := getPeerConfig(ip, publicKey, presharedKey, false)
		peerList = append(peerList, peerConfig)

		// Immediately update the interface. This will allow the new
		// config to connect. If we added a stale config to peerList
		// it is removed, essentially rotating this users's config.
		err = updateInterface(server, peerList)
		check(err)

		log.Printf("Added WireGuard peer %s %s %s", uid, ip, publicKey)
		log.Printf("Updated WireGuard interface %s", server.Interface)

	// Existing config and nothing to do - simply return the data.
	} else {
		ip = user[0].(string)
		publicKey = user[1].(string)
		privateKey = user[2].(string)
		presharedKey = user[3].(string)

		log.Printf("Found existing WireGuard peer %s %s %s", uid, ip, publicKey)
	}
	return nil, ip, publicKey, privateKey, presharedKey
}

// Periodically fetches user configs from Redis, and checks if a uid key has
// expired. Returns a peer list for updateInterface with expired configs, with
// the toRemove flag indicating that the server should remove the peer.
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

			// When both the base64 string and uid key exist, we'll
			// keep the config. When the uid key has expired, we'll
			// set toRemove to true, and remove the stale entries.
			if stringInSlice(s[3], keys) {
				// FIXME: Remove if all good. May be needed on
				// server restarts to fetch existing data tho.
				// peerConfig := getPeerConfig(s[0], s[1], s[2], false)
				// peerList = append(peerList, peerConfig)

				log.Printf("Keeping WireGuard peer %s %s %s", s[3], s[0], s[1])
			} else {
				// Remove stale base64 string from Redis.
				err = rc.SRem("users", b64).Err()
				check(err)

				// Free up IP.
				err = rc.SRem("usedIPs", s[0]).Err()
				check(err)

				// Add stale config to peerList with toRemove
				// set to true.
				peerConfig := getPeerConfig(s[0], s[1], s[2], true)
				peerList = append(peerList, peerConfig)

				log.Printf("Removing WireGuard peer %s %s %s", s[3], s[0], s[1])
			}
		}
	}

	return peerList
}
