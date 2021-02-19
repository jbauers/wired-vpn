package main

import (
	"context"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

// Expiry of Redis keys for WireGuard key rotation. We expire the "uid"
// key after the keyTTL value. Upon interface update, when the "uid"
// is missing, but present as part of the "users" SMEMBERS, we will
// free up the IP from "usedIPs" and remove the stale config.
var keyTTL = time.Duration(30 * time.Second)

// If a request comes in and the TTL for its "uid" key is less than this
// minTTL value, the WireGuard keys will be rotated. If no request comes
// in until the key is expired, it will be removed (as described above).
var minTTL = float64(10)

// Holds all peer information, whether that's a client or the server.
type Peer struct {
	Interface   string
	PublicKey   string
	PrivateKey  string
	PSK         string
	IP          string
	CIDR        string   `json:"cidr"`
	Endpoint    string   `json:"endpoint"`
	Port        int      `json:"port"`
	AllowedIPs  string   `json:"allowed_ips"`
	DNS         string   `json:"dns"`
	Groups      []string `json:"groups"`
	Access      bool
	Error       string
	RedisClient *redis.Client // Meh.
}

// Wrap []Peers in a struct for ServeHTTP.
type Servers struct {
	Peers []Peer
}

type Mail struct {
	Identity string `json:"identity"`
	Username string `json:"username"`
	Password string `json:"password"`
	Server   string `json:"server"`
	From     string `json:"from"`
	Notify   bool   `json:"notify"`
}

// Unmarshal our settings.json.
type Settings struct {
	Interfaces map[string]Peer `json:"interfaces"`
	Mail       Mail            `json:"mail"`
}

// Panic on error.
func check(e error) {
	if e != nil {
		log.Panic(e)
	}
}

// Returns true if string in slice.
func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Returns the WireGuard interface for a given group.
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

// Handles incoming HTTP requests. Expects that authentication has been taken
// care of upstream, as it takes the "X-Wired" headers passed from our proxy
// and decides what do with this client.
func (servers Servers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("/var/www/templates/wireguard.html"))
	w.Header().Add("Content-Type", "text/html")

	// Default to access denied.
	client := Peer{
		Access: false,
		Error:  "Access denied.",
	}

	// Get group and user from headers. We can get the server
	// this user belongs to from its group.
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

	// If both headers contain a valid value, continue.
	if wgInterface != "" && wgUser != "" {

		// All our servers are passed to ServeHTTP as peers.
		// Once we have the WireGuard interface for this
		// group, we use this server peer for this request.
		var server Peer
		for _, v := range servers.Peers {
			if wgInterface == v.Interface {
				server = v
			}
		}

		// Handle the user on this server. handleClient() decides
		// whether to rotate this user, add a new one, or return
		// exisiting data.
		err, clientIP, _, clientPrivateKey, clientPSK := handleClient(wgUser, server)

		// During handleClient() we might error, for example if
		// we run out of valid IP addresses. Render such an
		// error.
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
	// Read settings.
	var settings Settings
	s, err := ioutil.ReadFile("/settings.json")
	check(err)

	err = json.Unmarshal(s, &settings)
	check(err)

	// Init Redis
	rc := redisClient()

	// Prepare servers to be passed to ServeHTTP.
	var servers Servers
	for k, v := range settings.Interfaces {
		serverPrivateKey, serverPublicKey := initServer(k, v.CIDR, rc)
		check(err)
		server := Peer{
			Interface:   k,
			CIDR:        v.CIDR,
			Endpoint:    v.Endpoint,
			Port:        v.Port,
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

	// Periodically update all interfaces to remove expired
	// configurations.
	go func() {
		for true {
			time.Sleep(10 * time.Second)
			peerList := getPeerList(rc, settings.Mail)
			for _, server := range servers.Peers {
				updateInterface(server, peerList)
				log.Printf("Updated WireGuard interface %s", server.Interface)
			}
		}
	}()

	http.Handle("/", servers)
	log.Fatal(http.ListenAndServe(":9000", nil))
}
