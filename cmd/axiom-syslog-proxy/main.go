//
// Quick CLI test:
//    echo -n "udp message" | nc -u -w1 localhost 514
//    echo -n “tcp message” | nc -u -w1 localhost 601

package main

import (
	"flag"
	"log"
	"os"

	"github.com/axiomhq/axiom-syslog-proxy/server"
)

var (
	deploymentURL = os.Getenv("AXIOM_DEPLOYMENT_URL")
	ingestDataset = os.Getenv("AXIOM_INGEST_DATASET")
	ingestToken   = os.Getenv("AXIOM_INGEST_TOKEN")

	addrUDP = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
	addrTCP = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")
)

func main() {
	flag.Parse()

	if deploymentURL == "" {
		log.Fatal("missing AXIOM_DEPLOYMENT_URL")
	}
	if ingestDataset == "" {
		log.Fatal("missing AXIOM_INGEST_DATASET")
	}
	if ingestToken == "" {
		log.Fatal("missing AXIOM_INGEST_TOKEN")
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
		log.Fatalln(err)
	}
	srv.Run()
}
