package main

import (
	"flag"
	"log"
	"os"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/pkg/version"

	"github.com/axiomhq/axiom-syslog-proxy/server"
)

var (
	deploymentURL = os.Getenv("AXIOM_URL")
	ingestToken   = os.Getenv("AXIOM_TOKEN")
	ingestDataset = os.Getenv("AXIOM_DATASET")

	addrTCP = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")
	addrUDP = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
)

func main() {
	log.Print("starting axiom-syslog-proxy version ", version.Release())

	flag.Parse()

	if deploymentURL == "" {
		deploymentURL = axiom.CloudURL
	}
	if ingestToken == "" {
		log.Fatal("missing AXIOM_TOKEN")
	}
	if ingestDataset == "" {
		log.Fatal("missing AXIOM_DATASET")
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
