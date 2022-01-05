package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const subProtocol = "message-queue-v1"

var host = flag.String("host", "control", "API host")
var wsPort = flag.String("ws-port", "8080", "WS port on API host")
var registerPort = flag.String("register-port", "8081", "Register port on API host")
var wgInterface = flag.String("interface", "wg0", "WireGuard interface")
var wgEndpoint = flag.String("endpoint", "192.168.0.1", "WireGuard endpoint IP")
var wgPort = flag.Int("port", 51820, "WireGuard listen port")
var wgNetwork = flag.String("network", "10.100.0.1/24", "WireGuard network")
var wgAllowedIPs = flag.String("allowed-ips", "10.0.0.0/8", "WireGuard allowed IPs")
var wgDNS = flag.String("dns", "1.1.1.1", "WireGuard DNS")

func main() {
	flag.Parse()

	privateKey, err := wgtypes.GeneratePrivateKey()
	publicKey := privateKey.PublicKey().String()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	data := url.Values{
		"interface":  {*wgInterface},
		"endpoint":   {*wgEndpoint},
		"port":       {strconv.Itoa(*wgPort)},
		"pubkey":     {publicKey},
		"network":    {*wgNetwork},
		"allowedips": {*wgAllowedIPs},
		"dns":        {*wgDNS},
	}

	_, err = http.PostForm("http://"+*host+":"+*registerPort+"/register", data)
	check(err)

	u := url.URL{Scheme: "ws", Host: *host + ":" + *wsPort, Path: "/channel/" + *wgInterface}
	log.Printf("CONNECT %s", u.String())

	d := websocket.Dialer{Subprotocols: []string{subProtocol}}
	c, _, err := d.Dial(u.String(), nil)
	check(err)
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			check(err)
			log.Printf("RECV %s %s", *wgInterface, string(message))

			s := strings.Split(string(message), " ")
			action := s[0]
			ip := s[1]
			publicKey := s[2]
			presharedKey := s[3]
			uid := s[4]

			var peerList []wgtypes.PeerConfig
			switch action {
			case "ADD":
				peerConfig := getPeerConfig(ip, publicKey, presharedKey, false)
				peerList = append(peerList, peerConfig)
			case "DEL":
				peerConfig := getPeerConfig(ip, publicKey, presharedKey, true)
				peerList = append(peerList, peerConfig)
			default:
				return
			}

			err = updateInterface(privateKey, peerList)
			check(err)
			log.Printf("CONF %s %s %s %s %s %s", *wgInterface, action, ip, publicKey, presharedKey, uid)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.PongMessage, []byte(t.String()))
			check(err)
		case <-interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			check(err)
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
