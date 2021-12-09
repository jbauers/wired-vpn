package main

import (
	"encoding/base64"
	"log"
	"strings"

	"github.com/go-redis/redis/v8"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Assigns a free IP to a peer, returning the IP and an error. The usedIPs key
// keeps the currently assigned IPs as a set in Redis. Assigning or freeing up
// an IP is a matter of modifying this set.
func assignIP(cidr string, rc *redis.Client) (ip string, err error) {
	ips, err := rc.SMembers(ctx, "usedIPs").Result()
	check(err)

	ip, err = getAvailableIP(ips, cidr)
	if err != nil {
		return "", err
	}

	err = rc.SAdd(ctx, "usedIPs", ip).Err()
	check(err)

	return ip, nil
}

// Accepts the uid - an email - and "handles" this peer on the server. It will
// either just check Redis and simply return the data, or add a new peer config
// and update the server's interface. It also takes care of rotating configs
// that are expiring soon. In all cases, an error, the IP, and all keys for the
// peer are returned to be served by the web server.
func handleClient(uid string, clientPublicKey string, serverNetwork string, server Peer, redisClient *redis.Client) (err error, ip string, publicKey string, presharedKey string) {
	redisChannel := server.Interface
	redisUsers := server.Interface + "_users"

	rc := redisClient
	user, err := rc.HMGet(ctx, uid, "ip", "pubkey", "psk").Result()
	check(err)

	ttl, err := rc.TTL(ctx, uid).Result()
	check(err)

	// Either a new user, this user's config is expiring soon, or we got a new
	// public key. We need a new config and clean up stale configs for existing
	// users.
	if ttl.Seconds() < minTTL || user[0] == nil || user[1].(string) != clientPublicKey {
		// An existing user. Rotate the config.
		if user[0] != nil {
			staleIP := user[0].(string)
			stalePublicKey := user[1].(string)
			stalePresharedKey := user[2].(string)

			ref := staleIP + " " + stalePublicKey + " " + stalePresharedKey + " " + uid
			b64 := base64.StdEncoding.EncodeToString([]byte(ref))

			// Remove stale base64 string from Redis.
			err = rc.SRem(ctx, redisUsers, b64).Err()
			check(err)

			// Free up IP.
			err = rc.SRem(ctx, "usedIPs", staleIP).Err()
			check(err)

			// Publish on our channel.
			a := "DEL " + ref
			err = rc.Publish(ctx, redisChannel, a).Err()
			check(err)
			log.Printf("SEND %s %s", server.Interface, a)
		}

		// Generate new PSK and assign a free IP.
		psk, err := wgtypes.GenerateKey()
		check(err)
		presharedKey = psk.String()

		ip, err = assignIP(serverNetwork, rc)
		if err != nil {
			return err, "", "", ""
		}

		// Add the uid key with our IP and keys to Redis.
		peer := map[string]interface{}{
			"ip":     ip,
			"pubkey": clientPublicKey,
			"psk":    presharedKey,
		}
		err = rc.HMSet(ctx, uid, peer).Err()
		check(err)

		// Expire the uid key after keyTTL. When it is found missing
		// when getPeerList is called, the other Redis keys will also
		// be removed.
		err = rc.Expire(ctx, uid, keyTTL).Err()
		check(err)

		// Add IP, public key and pre-shared key as a base64 encoded
		// string to Redis.
		s := ip + " " + clientPublicKey + " " + presharedKey + " " + uid
		b64 := base64.StdEncoding.EncodeToString([]byte(s))

		err = rc.SAdd(ctx, redisUsers, b64).Err()
		check(err)

		// Use mullvad/message-queue here, and publish a message on this
		// channel. MQ will do the "heavy-lifting" for us and send a WSS
		// message on :8080/channel/peers. We can now simply have a
		// client listen on this URL and have it configure its interface
		// with this peer (WIP).
		a := "ADD " + s
		err = rc.Publish(ctx, redisChannel, a).Err()
		check(err)
		log.Printf("SEND %s %s", server.Interface, a)
	} else {
		ip = user[0].(string)
		publicKey = user[1].(string)
		presharedKey = user[2].(string)

		log.Printf("EXIST %s %s %s %s %s", server.Interface, ip, publicKey, presharedKey, uid)
	}
	ipCidrString := getIpCidrString(ip, serverNetwork)
	return nil, ipCidrString, publicKey, presharedKey
}

// Periodically fetches user configs from Redis, and checks if a uid key has
// expired. Returns a peer list for updateInterface with expired configs, with
// the toRemove flag indicating that the server should remove the peer.
func getPeerList(serverName string, newServer bool, rc *redis.Client) error {
	redisChannel := serverName
	redisUsers := serverName + "_users"

	users, err := rc.SMembers(ctx, redisUsers).Result()
	check(err)

	keys, err := rc.Keys(ctx, "*@*").Result()
	check(err)

	for _, b64 := range users {
		if b64 == "" {
			log.Print("No peers!")
		} else {
			decoded, _ := base64.StdEncoding.DecodeString(b64)
			s := strings.Split(string(decoded), " ")

			ip := s[0]
			publicKey := s[1]
			presharedKey := s[2]
			uid := s[3]

			// When both the base64 string and uid key exist, we'll
			// keep the config. When the uid key has expired, we'll
			// set toRemove to true, and remove the stale entries.
			if !stringInSlice(uid, keys) {
				// Remove stale base64 string from Redis.
				err = rc.SRem(ctx, redisUsers, b64).Err()
				check(err)

				// Free up IP.
				err = rc.SRem(ctx, "usedIPs", ip).Err()
				check(err)

				// Publish on our channel.
				a := "DEL " + string(decoded)
				err = rc.Publish(ctx, redisChannel, a).Err()
				check(err)
				log.Printf("SEND %s DEL %s %s %s %s", serverName, ip, publicKey, presharedKey, uid)
			} else if newServer {
				// Handle WireGguard server restarts properly.
				s := "ADD " + ip + " " + publicKey + " " + presharedKey + " " + uid
				err = rc.Publish(ctx, redisChannel, s).Err()
				check(err)
				log.Printf("SEND %s ADD %s %s %s %s", serverName, ip, publicKey, presharedKey, uid)
			}
		}
	}

	return err
}

func setServerInfo(serverInterface string, serverEndpoint string, serverPort string, serverPublicKey string, serverNetwork string, serverAllowedIPs string, serverDNS string, rc *redis.Client) (err error) {
	serverIP := strings.Split(serverNetwork, "/")[0]
	err = rc.SAdd(ctx, "usedIPs", serverIP).Err()
	check(err)

	peer := map[string]interface{}{
		"endpoint":   serverEndpoint,
		"port":       serverPort,
		"pubkey":     serverPublicKey,
		"network":    serverNetwork,
		"allowedips": serverAllowedIPs,
		"dns":        serverDNS,
	}
	err = rc.HMSet(ctx, serverInterface, peer).Err()
	log.Printf("SERVER: %s", serverInterface)

	return err
}

// Initialises the server, adding the server IP, public key and private key to
// Redis, and returning the keys as strings.
func getServerInfo(serverInterface string, rc *redis.Client) (serverEndpoint string, serverPort string, serverPublicKey string, serverNetwork string, serverAllowedIPs string, serverDNS string) {
	res, err := rc.HMGet(ctx, serverInterface, "endpoint", "port", "pubkey", "network", "allowedips", "dns").Result()
	check(err)

	if res[0] == nil {
		log.Printf("Server not found.")
	} else {
		serverEndpoint = res[0].(string)
		serverPort = res[1].(string)
		serverPublicKey = res[2].(string)
		serverNetwork = res[3].(string)
		serverAllowedIPs = res[4].(string)
		serverDNS = res[5].(string)
	}

	return serverEndpoint, serverPort, serverPublicKey, serverNetwork, serverAllowedIPs, serverDNS
}

// Returns a new Redis client.
func redisClient() (client *redis.Client) {
	client = redis.NewClient(&redis.Options{
		Addr:       "redis:6379",
		Password:   "pass",
		DB:         0,
		MaxRetries: 3,
	})
	return client
}
