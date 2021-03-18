package main

import (
	"flag"
	"log"
	"os"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/axiom-loki-proxy/version"
)

var (
	deploymentURL = os.Getenv("AXIOM_DEPLOYMENT_URL")
	accessToken   = os.Getenv("AXIOM_ACCESS_TOKEN")
	addrUDP       = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
	addrTCP       = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")
)

func main() {
	log.Print("starting axiom-loki-proxy version", version.Release())

	flag.Parse()

	if deploymentURL == "" {
		log.Fatal("missing AXIOM_DEPLOYMENT_URL")
	}
	if accessToken == "" {
		log.Fatal("missing AXIOM_ACCESS_TOKEN")
	}

	client, err := axiom.NewClient(deploymentURL, accessToken)
	if err != nil {
		log.Fatal(err)
	}

	// NOW WHAT?
}
