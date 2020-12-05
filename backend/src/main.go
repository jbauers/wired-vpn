package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

var serverInterface = os.Getenv("WG_SERVER_INTERFACE")
var serverCIDR = os.Getenv("WG_SERVER_CIDR")
var serverIP = os.Getenv("WG_SERVER_IP")
var serverEndpoint = os.Getenv("WG_SERVER_ENDPOINT")
var serverDNS = os.Getenv("WG_DNS")
var serverAllowedIPs = os.Getenv("WG_ALLOWED_IPS")

// Expiry of Redis keys for WireGuard key rotation. We expire the "uid"
// key after the keyTTL value. Upon interface update, when the "uid"
// is missing, but present as part of the "users" SMEMBERS, we will
// free up the IP from "usedIPs" and remove the stale config.
var keyTTL = time.Duration(600 * time.Second)

// If a request comes in and the TTL for its "uid" key is less than this
// minTTL value, the WireGuard keys will be rotated. If no request comes
// in until the key is expired, it will be removed (as described above).
var minTTL = float64(60)

type Peer struct {
	Interface   string
	Endpoint    string
	Port        int
	PublicKey   string
	PrivateKey  string
	PSK         string
	IP          string
	AllowedIPs  string
	DNS         string
	Access      bool
	Error       string
	RedisClient *redis.Client
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (server Peer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("/var/www/templates/wireguard.html"))
	w.Header().Add("Content-Type", "text/html")
	var client Peer
	for k, v := range r.Header {
		if k == "X-Wired-User" && v[0] != "" {
			err, clientIP, _, clientPrivateKey, clientPSK := handleClient(v[0], server)
			if err != nil {
				client = Peer{
					Access: false,
					Error:  err.Error(),
				}
			} else {
				client = Peer{
					Endpoint:   server.Endpoint,
					Port:       server.Port,
					PublicKey:  server.PublicKey,
					PrivateKey: clientPrivateKey,
					PSK:        clientPSK,
					IP:         clientIP,
					AllowedIPs: server.AllowedIPs,
					DNS:        server.DNS,
					Access:     true,
				}
			}
			break
		} else {
			client = Peer{
				Access: false,
				Error:  "Access denied.",
			}
		}
	}
	tmpl.Execute(w, client)
}

func main() {
	serverPort, err := strconv.Atoi(os.Getenv("WG_SERVER_PORT"))
	check(err)

	rc := redisClient()
	serverPrivateKey, serverPublicKey := initServer(rc)

	server := Peer{
		Interface:   serverInterface,
		Endpoint:    serverEndpoint,
		Port:        serverPort,
		PublicKey:   serverPublicKey,
		PrivateKey:  serverPrivateKey,
		AllowedIPs:  serverAllowedIPs,
		DNS:         serverDNS,
		RedisClient: rc,
	}

	log.Printf("---------------------- Backend ready -----------------------")
	log.Printf(" Interface: %s", server.Interface)
	log.Printf(" Network:   %s", serverCIDR)
	log.Printf(" Endpoint:  %s", server.Endpoint)
	log.Printf(" Port:      %d", server.Port)
	log.Printf(" PublicKey: %s", server.PublicKey)
	log.Printf("------------------------------------------------------------")

	go func() {
		for true {
			time.Sleep(10 * time.Second)

			peerList := getPeerList(rc)
			updateInterface(server, peerList)

			log.Printf("Updated WireGuard interface %s", server.Interface)
		}
	}()

	http.Handle("/", server)
	log.Fatal(http.ListenAndServe(":9000", nil))
}
