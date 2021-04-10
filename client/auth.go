package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// Start a blocking OIDC auth flow to obtain the peer from our proxied
// backend.
func authorizeUser(authorizationURL string, redirectURL string) (peer Peer) {
	// Start a web server to listen on a callback URL.
	server := &http.Server{Addr: redirectURL}
	defer server.Close()

	http.DefaultServeMux = new(http.ServeMux)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Incoming requests from our remote backend contain the base64
		// encoded JSON struct of our remote server in the peer argument.
		b64Peer := r.URL.Query().Get("peer")
		if b64Peer == "" {
			fmt.Println("Error: No peer found in query response")
			io.WriteString(w, "Error: No peer found in query response")

			cleanup(server)
			return
		}

		// Decode the b64 string and unmarshal our JSON.
		jsonPeer, _ := b64.StdEncoding.DecodeString(b64Peer)
		err := json.Unmarshal(jsonPeer, &peer)
		if err != nil {
			fmt.Println(err.Error())
			cleanup(server)
			return
		}
		peer.Access = true

		// Write a message for the end user. The CLI has all the data,
		// the auth flow was successful.
		io.WriteString(w, `
		<html>
		<head>
			<link rel="preconnect" href="https://fonts.gstatic.com">
			<link href="https://fonts.googleapis.com/css2?family=Montserrat:wght@100&display=swap" rel="stylesheet">
		</head>
		<body style="margin-top:50px;text-align:center;font-family:'Montserrat',sans-serif;">
			<h1>Login successful!</h1>
			<h2>Please return to the app. You may close this window.</h2>
		</body>
		</html>`)

		cleanup(server)
	})

	// Parse the redirect URL for the port number.
	u, err := url.Parse(redirectURL)
	if err != nil {
		fmt.Printf("Bad redirect URL: %s\n", err)
		os.Exit(1)
	}

	// Set up a listener on the redirect port.
	port := fmt.Sprintf(":%s", u.Port())
	l, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("Can't listen to port %s: %s\n", port, err)
		os.Exit(1)
	}

	// Open the URL for the auth flow cross-platform.
	var cmd *exec.Cmd
	switch platform := runtime.GOOS; platform {
	case "darwin":
		cmd = exec.Command("open", authorizationURL)
	case "linux":
		cmd = exec.Command("xdg-open", authorizationURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", authorizationURL)
	default:
		fmt.Printf("%s.\n", platform)
		os.Exit(1)
	}
	err = cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
	}

	// Start the blocking web server loop.
	// This will exit when the handler gets fired and calls server.Close(),
	// or when the channel times out.
	c := make(chan error, 1)
	go func() {
		e := server.Serve(l)
		c <- e
	}()

	select {
	case _ = <-c:
		break // Meh. HTTP server close is expected, update.
	case <-time.After(30 * time.Second):
		peer.Access = false
	}

	cleanup(server)
	return peer
}

// Closes the HTTP server.
func cleanup(server *http.Server) {
	// We run this as a goroutine so that this function falls through and
	// the socket to the browser gets flushed/closed before the server goes away.
	go server.Close()
}
