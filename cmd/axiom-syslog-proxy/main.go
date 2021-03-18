//
// Quick CLI test:
//    echo -n "udp message" | nc -u -w1 localhost 514
//    echo -n “tcp message” | nc -u -w1 localhost 601

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/logmanager"

	"github.com/axiomhq/axiom-syslog-proxy/input"
	"github.com/axiomhq/axiom-syslog-proxy/parser"
)

const (
	fieldApplication = "application"
	fieldHostname    = "hostname"
	fieldSeverity    = "severity"
	fieldText        = "message"
	fieldMetadata    = "metadata"
	fieldRemoteAddr  = "remoteAddress"
)

var (
	logger = logmanager.GetLogger("cmd.main")

	deploymentURL = os.Getenv("AXIOM_DEPLOYMENT_URL")
	ingestDataset = os.Getenv("AXIOM_INGEST_DATASET")
	ingestToken   = os.Getenv("AXIOM_INGEST_TOKEN")

	addrUDP = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
	addrTCP = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")

	axClient *axiom.Client
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

	var err error
	axClient, err = axiom.NewClient(deploymentURL, ingestToken)
	if err != nil {
		log.Fatal(err)
	}

	tcpParser := parser.New(onLogMessage)
	udpParser := parser.New(onLogMessage)

	closer, err := input.StartTCP(*addrTCP, tcpParser.WriteLine)
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close()

	closer, err = input.StartUDP(*addrUDP, udpParser.WriteLine)
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close()

	time.Sleep(time.Second * 60)
}

func onLogMessage(log *parser.Log) {
	ev := logToEvent(log)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	status, err := axClient.Datasets.IngestEvents(ctx, ingestDataset, axiom.IngestOptions{}, ev)
	if logger.IsError(err) {
	} else {
		logger.Trace("ingested %d event(s)", status.Ingested)
	}
}

func logToEvent(log *parser.Log) axiom.Event {
	ev := axiom.Event{}

	ev[axiom.TimestampField] = log.Timestamp
	ev[fieldSeverity] = strings.ToLower(parser.Severity(log.Severity).String())

	if log.Application != "" {
		ev[fieldApplication] = log.Application
	}
	if log.Hostname != "" {
		ev[fieldHostname] = log.Hostname
	}
	if log.Text != "" {
		ev[fieldText] = log.Text
	}
	if log.RemoteAddr != "" {
		ev[fieldRemoteAddr] = log.RemoteAddr
	}
	if len(log.Metadata) > 0 {
		ev[fieldMetadata] = log.Metadata
	}

	return ev
}
