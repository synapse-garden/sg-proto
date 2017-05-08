package main

import (
	"log"
	"net/http"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/rest"

	"github.com/boltdb/bolt"
)

func serveInsecure(
	db *bolt.DB,
	apiKey auth.Token,
	addr, port string,
	source rest.SourceInfo,
) {
	router, cleanups, err := rest.Bind(db, source, apiKey)
	defer cleanups.Cleanup()
	if err != nil {
		log.Fatalf("failed to bind on DB: %s", err.Error())
	}

	log.Printf("SG Proto serving INSECURELY at http://%s%s", addr, port)
	log.Fatal(http.ListenAndServe(addr+port, router))
}

func serveSecure(
	db *bolt.DB,
	apiKey auth.Token,
	addr, port, cert, key string,
	source rest.SourceInfo,
) {
	router, cleanups, err := rest.Bind(db, source, apiKey)
	defer cleanups.Cleanup()
	if err != nil {
		log.Fatalf("failed to bind on DB: %s", err.Error())
	}

	log.Printf("SG Proto serving at https://%s%s", addr, port)
	log.Fatal(http.ListenAndServeTLS(
		addr+port,
		cert, key,
		router,
	))
}

func devServeInsecure(
	db *bolt.DB,
	apiKey auth.Token,
	addr, port string,
	source rest.SourceInfo,
) {
	router, cleanups, err := rest.Bind(db, source, apiKey)
	defer cleanups.Cleanup()
	if err != nil {
		log.Fatalf("failed to bind on DB: %s", err.Error())
	}

	log.Printf("SG Proto serving INSECURELY in dev mode at "+
		"http://%s%s", addr, port)
	log.Fatal(http.ListenAndServe(addr+port, router))
}

func devServeSecure(
	db *bolt.DB,
	apiKey auth.Token,
	addr, port, cert, key string,
	source rest.SourceInfo,
) {
	router, cleanups, err := rest.Bind(db, source, apiKey)
	defer cleanups.Cleanup()
	if err != nil {
		log.Fatalf("failed to bind on DB: %s", err.Error())
	}

	log.Printf("SG Proto hosting in dev mode at https://%s%s",
		addr, port)
	log.Fatal(http.ListenAndServeTLS(
		addr+port,
		cert, key,
		router,
	))
}
