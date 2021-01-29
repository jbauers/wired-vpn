package main

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

// Expiry of Redis keys for WireGuard key rotation. We expire the "uid"
// key after the keyTTL value. Upon interface update, when the "uid"
// is missing, but present as part of the "users" SMEMBERS, we will
// free up the IP from "usedIPs" and remove the stale config.
var keyTTL = time.Duration(30 * time.Second)

// If a request comes in and the TTL for its "uid" key is less than this
// minTTL value, the WireGuard keys will be rotated. If no request comes
// in until the key is expired, it will be removed (as described above).
var minTTL = float64(10)

// FIXME: Clean up redundant stuff.
type Peer struct {
	Interface   string
	CIDR        string
	Endpoint    string
	Port        int
	PublicKey   string
	PrivateKey  string
	PSK         string
	IP          string
	AllowedIPs  string
	DNS         string
	Groups      []string
	Access      bool
	Error       string
	RedisClient *redis.Client // Meh.
}

type Servers struct {
	Peers []Peer
}

type Settings struct {
	Interfaces map[string]Interface `json:"interfaces"`
}

type Interface struct {
	CIDR       string   `json:"cidr"`
	Endpoint   string   `json:"endpoint"`
	Port       string   `json:"port"`
	AllowedIPs string   `json:"allowed_ips"`
	DNS        string   `json:"dns"`
	Groups     []string `json:"groups"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func getGroupInterface(peers []Peer, group string) string {
	for _, p := range peers {
		for _, g := range p.Groups {
			if group == g {
				return p.Interface
			}
		}
	}
	return ""
}

func (servers Servers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("/var/www/templates/wireguard.html"))
	w.Header().Add("Content-Type", "text/html")

	client := Peer{
		Access: false,
		Error:  "Access denied.",
	}

	// Get interface and user from header.
	headers := make(map[string]interface{})
	for k, v := range r.Header {
		headers[k] = string(v[0])
	}

	wgInterface := ""
	if value, ok := headers["X-Wired-Group"]; ok {
		wgInterface = getGroupInterface(servers.Peers, value.(string))
	}

	wgUser := ""
	if value, ok := headers["X-Wired-User"]; ok {
		wgUser = value.(string)
	}

	// If both contain a valid value, continue.
	if wgInterface != "" && wgUser != "" {
		var server Peer
		for _, v := range servers.Peers {
			if wgInterface == v.Interface {
				server = v
			}
		}
		err, clientIP, _, clientPrivateKey, clientPSK := handleClient(wgUser, server)
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
	}
	tmpl.Execute(w, client)
}

func main() {
	var settings Settings
	s, err := ioutil.ReadFile("/settings.json")
	check(err)

	err = json.Unmarshal(s, &settings)
	check(err)

	// Init Redis
	rc := redisClient()

	var servers Servers
	for k, v := range settings.Interfaces {
		serverPrivateKey, serverPublicKey := initServer(k, v.CIDR, rc)
		serverPort, err := strconv.Atoi(v.Port)
		check(err)
		server := Peer{
			Interface:   k,
			CIDR:        v.CIDR,
			Endpoint:    v.Endpoint,
			Port:        serverPort,
			PublicKey:   serverPublicKey,
			PrivateKey:  serverPrivateKey,
			AllowedIPs:  v.AllowedIPs,
			Groups:      v.Groups,
			DNS:         v.DNS,
			RedisClient: rc,
		}
		servers.Peers = append(servers.Peers, server)
	}

	for _, server := range servers.Peers {
		log.Printf("---------------------- Backend ready -----------------------")
		log.Printf(" Interface:  %s", server.Interface)
		log.Printf(" Network:    %s", server.CIDR)
		log.Printf(" Endpoint:   %s", server.Endpoint)
		log.Printf(" Port:       %d", server.Port)
		log.Printf(" PublicKey:  %s", server.PublicKey)
		log.Printf(" DNS:        %s", server.DNS)
		log.Printf(" Groups:     %s", server.Groups)
		log.Printf(" AllowedIPs: %s", server.AllowedIPs)
		log.Printf("------------------------------------------------------------")
	}

	go func() {
		for true {
			time.Sleep(10 * time.Second)

			// FIXME: Improve.
			peerList := getPeerList(rc)
			for _, server := range servers.Peers {
				updateInterface(server, peerList)
				log.Printf("Updated WireGuard interface %s", server.Interface)
			}
		}
	}()

	http.Handle("/", servers)
	log.Fatal(http.ListenAndServe(":9000", nil))
}
