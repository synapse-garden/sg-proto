package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/synapse-garden/sg-proto/rest"

	"github.com/boltdb/bolt"
)

// CLI flags
var (
	Address  = flag.String("addr", "127.0.0.1", "the address to host on")
	Port     = flag.String("port", ":8080", "the port to listen on")
	DBAddr   = flag.String("db", "my.db", "the database to use")
	CertFile = flag.String("cert", "", "the certificate file to use")
	KeyFile  = flag.String("key", "cert.key", "the certificate key to use")
	ConfFile = flag.String("cfg", "conf.toml", "the config file to use")

	SourceLocation = flag.String("source", "https://github.com/synapse-garden/sg-proto", "where the source is hosted")

	Dev = flag.Bool("dev", false, "start in developer mode")
)

const (
	SourceLicense = "Affero GPL V3"
	Licensee      = "SynapseGarden 2016"
)

func main() {
	flag.Parse()

	DB, err := bolt.Open(*DBAddr, 0600, nil)
	if err != nil {
		log.Fatalf("unable to open Bolt database: %s", err.Error())
	}

	source := &rest.SourceInfo{
		Location:   *SourceLocation,
		License:    SourceLicense,
		LicensedTo: Licensee,
	}

	switch {
	case *SourceLocation == "" && !*Dev:
		log.Fatal("must provide a source location using -source")
	case *CertFile == "":
		log.Printf(
			"SG Proto hosting INSECURELY at http://%s%s",
			*Address, *Port,
		)
		router, err := rest.Bind(source, DB)
		if err != nil {
			log.Fatalf("failed to bind on DB: %s", err.Error())
		}
		log.Fatal(http.ListenAndServe(*Address+*Port, router))
	}

	log.Printf(
		"SG Proto hosting securely at https://%s%s",
		*Address, *Port,
	)
	router, err := rest.Bind(source, DB)
	if err != nil {
		log.Fatalf("failed to bind on DB: %s", err.Error())
	}

	log.Fatal(http.ListenAndServeTLS(
		*Address+*Port,
		*CertFile,
		*KeyFile,
		router,
	))
}
