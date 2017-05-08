package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/client"
)

var (
	host       = flag.String("host", "127.0.0.1", "the host to connect to")
	port       = flag.String("port", ":8080", "the port to connect on")
	acceptCert = flag.String("acceptCert", "", "a custom cert to accept")

	secure = flag.Bool("secure", true, "whether to connect securely")
)

func main() {
	flag.Parse()

	if c := *acceptCert; c != "" {
		if !strings.HasSuffix(c, "pem") {
			log.Fatalf("custom cert must be a .pem file")
		}
		if certBytes, err := ioutil.ReadFile(c); err != nil {
			log.Fatalf("failed to read custom cert: %s", err.Error())
		} else if err = client.SetCustomCert(certBytes); err != nil {
			log.Fatalf("failed to set custom cert: %s", err.Error())
		}
	}

	scheme := "http"
	if *secure {
		scheme = "https"
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
