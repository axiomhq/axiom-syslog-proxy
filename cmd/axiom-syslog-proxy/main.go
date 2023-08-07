package main

import (
	"context"
	"flag"
	"os"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/pkg/cmd"
	"go.uber.org/zap"

	"github.com/axiomhq/axiom-syslog-proxy/server"
)

var (
	addrTCP = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")
	addrUDP = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
)

func main() {
	cmd.Run("axiom-syslog-proxy", run,
		cmd.WithRequiredEnvVars("AXIOM_DATASET"),
		cmd.WithValidateAxiomCredentials(),
	)
}

func run(_ context.Context, _ *zap.Logger, client *axiom.Client) error {
	flag.Parse()

	config := &server.Config{
		Dataset: os.Getenv("AXIOM_DATASET"),
		AddrUDP: *addrUDP,
		AddrTCP: *addrTCP,
	}

	srv, err := server.NewServer(client, config)
	if err != nil {
		return cmd.Error("create server", err)
	}

	srv.Run()

	return nil
}
