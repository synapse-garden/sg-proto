package main

import (
	"bufio"
	"flag"
	"log"
	"net/url"
	"os"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/client"
)

var (
	ticket = flag.String("ticket", "", "a ticket to incept a new user")
	host   = flag.String("host", "127.0.0.1", "the host to connect to")
	port   = flag.String("port", ":8080", "the port to connect on")
	user   = flag.String("user", "bob", "the user to chat with")

	secure = flag.Bool("secure", true, "whether to connect securely")
)

func main() {
	flag.Parse()

	scheme := "https"
	if !*secure {
		scheme = "http"
	}

	hostURL, err := url.Parse(scheme + "://" + *host + *port)
	if err != nil {
		log.Fatalf("error parsing host URL: %s", err.Error())
	}

	cli := &client.Client{
		Backend: hostURL,
		State: client.State{
			Session: new(auth.Session),
			Writer:  bufio.NewWriter(os.Stdout),
			Scanner: bufio.NewScanner(os.Stdin),
		},
	}
	if err := RunWindow(cli); err != nil {
		log.Fatalf("Synapse crashed: %s", err.Error())
	}
}
