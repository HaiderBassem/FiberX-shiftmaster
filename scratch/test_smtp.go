package main

import (
	"fmt"
	"net/smtp"
	"errors"
)

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		fmt.Printf("Server sent: %q\n", string(fromServer))
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unknown from server: " + string(fromServer))
		}
	}
	return nil, nil
}

func main() {
	host := "smtp.office365.com"
	port := 587
	user := "support@fiberx.iq"
	pass := "P*880660116019ow"
	from := "support@fiberx.iq"
	to := []string{"support@fiberx.iq"}

	addr := fmt.Sprintf("%s:%d", host, port)
	auth := LoginAuth(user, pass)

	msg := []byte("To: support@fiberx.iq\r\n" +
		"From: " + from + "\r\n" +
		"Subject: Test Email\r\n" +
		"\r\n" +
		"This is a test email.\r\n")

	fmt.Println("Sending email...")
	err := smtp.SendMail(addr, auth, from, to, msg)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Email sent successfully!")
	}
}
