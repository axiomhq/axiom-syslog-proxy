package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/pkg/version"

	"github.com/axiomhq/axiom-syslog-proxy/server"
)

const (
	exitOK int = iota
	exitConfig
	exitInternal
)

var (
	addrTCP = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")
	addrUDP = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
)

func main() {
	os.Exit(Main())
}

func Main() int {
	// Export `AXIOM_TOKEN` and `AXIOM_ORG_ID` for Axiom Cloud
	// Export `AXIOM_URL` and `AXIOM_TOKEN` for Axiom Selfhost

	log.Print("starting axiom-syslog-proxy version ", version.Release())

	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		os.Kill,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer cancel()

	dataset := os.Getenv("AXIOM_DATASET")
	if dataset == "" {
		log.Print("AXIOM_DATASET is required")
		return exitConfig
	}

	client, err := axiom.NewClient()
	if err != nil {
		log.Print(err)
		return exitConfig
	} else if err = client.ValidateCredentials(ctx); err != nil {
		log.Print(err)
		return exitConfig
	}

	config := &server.Config{
		Dataset: dataset,
		AddrUDP: *addrUDP,
		AddrTCP: *addrTCP,
	}

	srv, err := server.NewServer(client, config)
	if err != nil {
		log.Print(err)
		return exitInternal
	}

	srv.Run()

	return exitOK
}
