package main

import (
	"flag"
	"log"
	"os"

	"github.com/axiomhq/pkg/version"

	"github.com/axiomhq/axiom-syslog-proxy/server"
)

var (
	deploymentURL = os.Getenv("AXIOM_DEPLOYMENT_URL")
	ingestToken   = os.Getenv("AXIOM_ACCESS_TOKEN")
	ingestDataset = os.Getenv("AXIOM_INGEST_DATASET")

	addrTCP = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")
	addrUDP = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
)

func main() {
	log.Print("starting axiom-syslog-proxy version ", version.Release())

	flag.Parse()

	if deploymentURL == "" {
		log.Fatal("missing AXIOM_DEPLOYMENT_URL")
	}
	if ingestToken == "" {
		log.Fatal("missing AXIOM_ACCESS_TOKEN")
	}
	if ingestDataset == "" {
		log.Fatal("missing AXIOM_INGEST_DATASET")
	}

	config := &server.Config{
		URL:     deploymentURL,
		Dataset: ingestDataset,
		Token:   ingestToken,
		AddrUDP: *addrUDP,
		AddrTCP: *addrTCP,
	}

	srv, err := server.NewServer(config)
	if err != nil {
		log.Fatal(err)
	}

	srv.Run()
}
