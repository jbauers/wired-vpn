package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"time"
)

// Use SSL on 465 instead of STARTTLS on 587:
// - https://stackoverflow.com/questions/57063411/go-smtp-unable-to-send-email-through-gmail-getting-eof/57076378
// - https://gist.github.com/chrisgillis/10888032
func sendMail(receiver string, m Mail, msg string) {

	from := mail.Address{"", m.Username}
	to := mail.Address{"", receiver}
	subj := "IMPORTANT - VPN access"
	body := "Hi there -\n\n" + msg + "\n\nThat's all!"

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = m.From
	headers["To"] = to.String()
	headers["Date"] = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	headers["Subject"] = subj

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := m.Server

	host, _, _ := net.SplitHostPort(servername)
	auth := smtp.PlainAuth(m.Identity, m.Username, m.Password, host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}

	conn, err := tls.Dial("tcp", servername, tlsconfig)
	if err != nil {
		log.Panic(err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Panic(err)
	}

	// Auth
	if err = c.Auth(auth); err != nil {
		log.Panic(err)
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		log.Panic(err)
	}

	if err = c.Rcpt(to.Address); err != nil {
		log.Panic(err)
	}

	// Data
	w, err := c.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Panic(err)
	}

	err = w.Close()
	if err != nil {
		log.Panic(err)
	}

	c.Quit()
}
