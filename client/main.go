package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/go-ping/ping"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// We'll add a value when compiling.
var endpoint string

type Peer struct {
	Interface  string   `json:"interface"`
	PublicKey  string   `json:"public_key"`
	PrivateKey string   `json:"private_key"`
	PSK        string   `json:"psk"`
	IP         string   `json:"ip"`
	CIDR       string   `json:"cidr"`
	Endpoint   string   `json:"endpoint"`
	Port       int      `json:"port"`
	AllowedIPs string   `json:"allowed_ips"`
	DNS        string   `json:"dns"`
	Groups     []string `json:"groups"`
	Access     bool     `json:"access"`
	Error      string   `json:"error"`
}

func apiCall(publicKey string) (peer Peer) {
	// Plain HTTP is a bad idea, and most IdPs will complain unless it's
	// localhost. Our scripts set up SSL certs, and may require some
	// /etc/hosts magic for local testing.
	authorizationURL := "https://" + endpoint + "/?public_key=" + publicKey

	// If you change this, you need to change it on the server side as well.
	// This is a callback and should be ok.
	redirectURL := "http://localhost:9999/"

	// This starts a blocking OIDC auth flow. Note that the actual auth happens
	// between our remote server and the IdP. The response from the backend is
	// passed as a base64 encoded JSON struct in a request parameter to an HTTP
	// server this CLI spawns locally. There's a timeout after 15s.
	peer = authorizeUser(authorizationURL, redirectURL)
	return peer
}

func getAllowedIP(ip string) []net.IPNet {
	_, ipnet, _ := net.ParseCIDR(ip)
	return []net.IPNet{*ipnet}
}

// FIXME: Probably better to use net and not stitch strings together...
func getServerPrivateIP(ip string) string {
	i := strings.Split(ip, "/")[0] // Remove CIDR.
	s := strings.Split(i, ".")
	peerIPStrings := []string{s[0], s[1], s[2], "1"} // RIP CIDR larger than a /24
	peerIP := strings.Join(peerIPStrings, ".")
	return peerIP
}

func pingServer(host string) string {
	// Use go-ping so we can specifically add CAP_NET_RAW
	// to our binary and avoid sudo. Also doesn't require
	// elevation on Windows.
	pinger, err := ping.NewPinger(host)
	if err != nil {
		fmt.Println(err.Error())
		return "Fatal error"
	}
	pinger.SetPrivileged(true) // Needed for Windows, but doesn't require elevation.
	pinger.Timeout = 3 * time.Second
	err = pinger.Run() // Blocks until finished, but we set the timeout. Count would wait.
	if err != nil {
		fmt.Println(err.Error())
		return "Fatal error"
	}
	stats := pinger.Statistics()
	if !(stats.PacketsRecv > 0) {
		return fmt.Sprintf("%s, %d transmitted, %v%% loss",
			host, stats.PacketsSent, stats.PacketLoss)
	}
	return host
}

func updateInterface(wgInterface string, peer Peer) string {
	// Configure Linux networking. Tenus is used as it works
	// with CAP_NET_ADMIN permissions, and we can avoid
	// running the CLI as root, which causes other issues.
	// FIXME: Cross-platform support.
	err, wired := configureInterface(wgInterface, peer)
	if err != nil {
		return err.Error()
	}

	res := setInterfaceUp(wired)

	return res
}

func ticker(message *widget.TextGrid, done chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	i := 0
	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				i++
				s := fmt.Sprintf(`

                                
        Connecting: %ds


`, 30-i)
				message.SetText(s)
			}
		}
	}()
}

func main() {
	notConnectedMsg := `


         Not connected          


`
	connectingMsg := `


         Connecting...          


`

	// Generate new keys on start.
	// FIXME: Do we still need them as strings?
	keyPair, _ := wgtypes.GeneratePrivateKey()
	privateKey := keyPair.String()
	publicKey := keyPair.PublicKey().String()

	// Apply private key.
	wc, _ := wgctrl.New()
	wgInterface := "wired0" // FIXME: Don't hardcode?
	wgPrivateKey, _ := wgtypes.ParseKey(privateKey)
	config := wgtypes.Config{
		PrivateKey: &wgPrivateKey,
	}
	wc.ConfigureDevice(wgInterface, config)

	light := theme.LightTheme()
	a := app.New()
	a.Settings().SetTheme(light)
	w := a.NewWindow("Wired")

	message := widget.NewTextGridFromString(notConnectedMsg)

	var peer Peer
	var connecting bool

	button := widget.NewButton("Connect", func() {})
	button.ExtendBaseWidget(button)
	button.OnTapped = func() {
		message.SetText(connectingMsg)
		connecting = true
		button.Disable()
		button.SetText("Waiting for response")

		done := make(chan bool)
		ticker(message, done)
		msg := notConnectedMsg

		// This will run until successful, or block until we time
		// out. We have disabled our button and started a timer.
		if peer = apiCall(publicKey); peer.Access == true {
			// Configure our local interface.
			peer.PrivateKey = privateKey
			updateInterface(wgInterface, peer)

			// Ping our endpoint.
			peerIP := getServerPrivateIP(peer.IP)
			if msg = pingServer(peerIP); msg == peerIP {
				msg = fmt.Sprintf(`            Success!            

 Peer:  %s
 IP:    %s
 Route: %s
 DNS:   %s
`,
					peer.Endpoint, peer.IP, peer.AllowedIPs, peer.DNS)
				button.SetText("Reconnect")
			}
		} else {
			button.SetText("Connect")
		}

		done <- true
		connecting = false
		button.Enable()
		button.Refresh()
		message.SetText(msg)
	}

	w.SetContent(container.NewVBox(
		message,
		button,
	))

	go func() {
		for true {
			time.Sleep(10 * time.Second)
			if peer.IP != "" {
				peerIP := getServerPrivateIP(peer.IP)
				res := pingServer(peerIP)
				fmt.Println(res)
				if res != peerIP && connecting != true {
					message.SetText(notConnectedMsg)
				}
			}
			w.Canvas().Refresh(w.Content())
		}
	}()

	w.ShowAndRun()
}
