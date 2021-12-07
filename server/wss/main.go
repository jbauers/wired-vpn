package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const subProtocol = "message-queue-v1"

var addr = flag.String("addr", "api:8080", "http service address")
var wgInterface = flag.String("interface", "wg0", "WireGuard interface")
var wgPort = flag.Int("port", 51820, "WireGuard listen port")
var wgNetwork = flag.String("network", "10.100.0.1/24", "WireGuard network")

func check(e error) {
	if e != nil {
		log.Panic(e)
	}
}

// The allowed IPv4 for clients to be added to the server may only be a /32,
// but wgctrl expects a list. This function returns the list with a single
// entry from an IP string.
func getAllowedIP(ip string) []net.IPNet {
	_, ipnet, err := net.ParseCIDR("0.0.0.0/32")
	check(err)

	network := *ipnet
	network.IP = net.ParseIP(ip)

	return []net.IPNet{network}
}

// Takes the IP, public key, pre-shared key as strings, and a bool whether the
// peer should be removed or added to the interface, and returns the wgtypes
// peer config for this peer. This config is then applied as part of
// updateInterface, which expects a list of these peer configs.
func getPeerConfig(ip string, publicKey string, presharedKey string, toRemove bool) (peerConfig wgtypes.PeerConfig) {
	pub, err := wgtypes.ParseKey(publicKey)
	check(err)

	psk, err := wgtypes.ParseKey(presharedKey)
	check(err)

	allowedIPs := getAllowedIP(ip)

	peerConfig = wgtypes.PeerConfig{
		PublicKey:         pub,
		PresharedKey:      &psk,
		Remove:            toRemove,
		AllowedIPs:        allowedIPs,
		ReplaceAllowedIPs: false,
	}

	return peerConfig
}

// Takes a list of peer configs and applies the config to the server specified.
// Peers are not replaced, instead the peer configs indicate whether a peer
// should be removed or appended to the server. Rotating peers works by passing
// both the stale and new configs as part of the peer list, with the toRemove
// flag indicating what to do (see getPeerConfig).
func updateInterface(privateKey wgtypes.Key, peerList []wgtypes.PeerConfig) error {
	wc, err := wgctrl.New()
	check(err)

	port := *wgPort

	config := wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &port,
		Peers:        peerList,
		ReplacePeers: false,
	}

	err = wc.ConfigureDevice(*wgInterface, config)
	return err
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	privateKey, err := wgtypes.GeneratePrivateKey()
	publicKey := privateKey.PublicKey().String()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/channel/peers"}
	log.Printf("connecting to %s", u.String())

	d := websocket.Dialer{Subprotocols: []string{subProtocol}}
	c, _, err := d.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", string(message))
			s := strings.Split(string(message), " ")
			action := s[0]
			ip := s[1]
			publicKey := s[2]
			presharedKey := s[3]
			var peerList []wgtypes.PeerConfig
			if action == "ADD" {
				peerConfig := getPeerConfig(ip, publicKey, presharedKey, false)
				peerList = append(peerList, peerConfig)
			} else if action == "DEL" {
				peerConfig := getPeerConfig(ip, publicKey, presharedKey, true)
				peerList = append(peerList, peerConfig)
			} else {
				log.Fatal("unsupported action, aborting")
			}
			err = updateInterface(privateKey, peerList)
			if err != nil {
				log.Println("couldn't update interface:", err)
				return
			}
		}
	}()

	go func() {
		publicHandler := func(w http.ResponseWriter, req *http.Request) {
			io.WriteString(w, publicKey+" "+*wgNetwork)
		}

		http.HandleFunc("/public", publicHandler)
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.PongMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
