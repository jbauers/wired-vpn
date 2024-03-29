package main

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var ctx = context.Background()

// Expiry of Redis keys for WireGuard key rotation. We expire the "uid"
// key after the keyTTL value. Upon interface update, when the "uid"
// is missing, but present as part of the "users" SMEMBERS, we will
// free up the IP from "usedIPs" and remove the stale config.
var keyTTL = time.Duration(1 * time.Minute)

// If a request comes in and the TTL for its "uid" key is less than this
// minTTL value, the WireGuard keys will be rotated. If no request comes
// in until the key is expired, it will be removed (as described above).
var minTTL = float64(10)

// Holds all peer information, whether that's a client or the server.
type Peer struct {
	Interface  string   `json:"interface"`
	PublicKey  string   `json:"public_key"`
	PrivateKey string   `json:"private_key"`
	PSK        string   `json:"psk"`
	IP         string   `json:"ip"`
	CIDR       string   `json:"cidr"`
	Endpoint   string   `json:"endpoint"`
	Port       string   `json:"port"`
	AllowedIPs string   `json:"allowed_ips"`
	DNS        string   `json:"dns"`
	Groups     []string `json:"groups"`
	Access     bool     `json:"access"`
	Error      string   `json:"error"`
}

// Wrap []Peers in a struct for ServeHTTP.
type Servers struct {
	Peers       []Peer
	RedisClient *redis.Client
}

// Unmarshal our settings.json.
type Settings struct {
	Interfaces map[string]Peer `json:"interfaces"`
}

// Handles incoming HTTP requests. Expects that authentication has been taken
// care of upstream, as it takes the "X-Wired" headers passed from our proxy
// and decides what do with this client.
func (servers Servers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Default to access denied.
	client := Peer{
		Access: false,
		Error:  "Access denied.",
	}

	// Get user, their group and public key from headers. Only the
	// public key can be provided by the client, which we sanity
	// check before doing anything. The rest happens between our
	// proxy and the IdP.
	headers := make(map[string]interface{})
	for k, v := range r.Header {
		headers[k] = string(v[0])
	}

	// We can get the server this user belongs to from their OIDC group
	// as mapped in /settings.json. The group is added to this header
	// by our proxy as provided by the IdP, so we shouldn't need further
	// validation.
	wgInterface := ""
	if value, ok := headers["X-Wired-Group"]; ok {
		wgInterface = getGroupInterface(servers.Peers, value.(string))
	}

	// The user header is also added by our proxy from the IdP response.
	wgUser := ""
	if value, ok := headers["X-Wired-User"]; ok {
		wgUser = value.(string)
	}

	// Sanity check the provided key. If we can apply it, we don't care
	// if the user willingly provided a wrong one. Worst case they can't
	// connect.
	wgPublicKey := ""
	if value, ok := headers["X-Wired-Public-Key"]; ok {
		wgPublicKey = value.(string)
		_, e := wgtypes.ParseKey(wgPublicKey)
		if e != nil {
			wgPublicKey = ""
		}
	}

	// After validation, if all headers contain a value, continue.
	if wgInterface != "" && wgUser != "" && wgPublicKey != "" {

		// All our servers are passed to ServeHTTP as peers.
		// Once we have the WireGuard interface for this
		// group, we have a server to use for this request.
		var server Peer
		for _, s := range servers.Peers {
			if wgInterface == s.Interface {
				server = s
			}
		}

		// Handle the user on this server. handleClient() decides
		// whether to rotate this user, add a new one, or return
		// exisiting data.
		serverEndpoint, serverPort, serverPublicKey, serverNetwork, serverAllowedIPs, serverDNS := getServerInfo(server.Interface, servers.RedisClient)
		err, clientIP, _, clientPSK := handleClient(wgUser, wgPublicKey, serverNetwork, server, servers.RedisClient)

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
				Endpoint:   serverEndpoint,
				Port:       serverPort,
				PublicKey:  serverPublicKey,
				PSK:        clientPSK,
				IP:         clientIP,
				AllowedIPs: serverAllowedIPs,
				DNS:        serverDNS,
				Access:     true,
			}
		}
	}
	jsonPeer, err := json.Marshal(client)
	check(err)
	b64Peer := b64.StdEncoding.EncodeToString([]byte(jsonPeer))

	// Redirect back to CLI.
	w.Header().Set("Location", "http://localhost:9999/?peer="+b64Peer)
	w.WriteHeader(http.StatusFound)
}

func main() {
	// Read settings.
	var settings Settings
	s, err := ioutil.ReadFile("/settings.json")
	check(err)

	err = json.Unmarshal(s, &settings)
	check(err)

	// Init Redis.
	rc := redisClient()

	// Prepare servers to be passed to ServeHTTP.
	var servers Servers
	for iface, setting := range settings.Interfaces {
		server := Peer{
			Interface: iface,
			Groups:    setting.Groups,
		}
		servers.Peers = append(servers.Peers, server)
	}

	go func() {
		registerHandler := func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				if err := r.ParseForm(); err != nil {
					io.WriteString(w, "Form parse error.")
					return
				}

				serverInterface := r.FormValue("interface")
				serverEndpoint := r.FormValue("endpoint")
				serverPort := r.FormValue("port")
				serverPublicKey := r.FormValue("pubkey")
				serverNetwork := r.FormValue("network")
				serverAllowedIPs := r.FormValue("allowedips")
				serverDNS := r.FormValue("dns")

				err = setServerInfo(serverInterface, serverEndpoint, serverPort, serverPublicKey, serverNetwork, serverAllowedIPs, serverDNS, rc)
				check(err)

				io.WriteString(w, "ok")

				// Publish currently stored users from Redis.
				go func() {
					time.Sleep(2 * time.Second)
					err := getPeerList(serverInterface, true, rc)
					check(err)
				}()
			default:
				io.WriteString(w, "Sorry, only POST supported.")
			}
		}

		http.HandleFunc("/register", registerHandler)
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	// Periodically update all interfaces to remove expired
	// configurations.
	go func() {
		for true {
			time.Sleep(10 * time.Second)
			for name, _ := range settings.Interfaces {
				err := getPeerList(name, false, rc)
				check(err)
			}
		}
	}()

	servers.RedisClient = rc
	http.Handle("/", servers)
	log.Fatal(http.ListenAndServe(":9000", nil))
}
