package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/axiomhq/axiom-syslog-proxy/input"
	"github.com/axiomhq/axiom-syslog-proxy/parser"
)

var (
	deploymentURL = os.Getenv("AXIOM_DEPLOYMENT_URL")
	accessToken   = os.Getenv("AXIOM_ACCESS_TOKEN")
	addrUDP       = flag.String("addr-udp", ":514", "Listen address <ip>:<port>")
	addrTCP       = flag.String("addr-tcp", ":601", "Listen address <ip>:<port>")
)

func main() {
	flag.Parse()

	/*
		if deploymentURL == "" {
			log.Fatal("missing AXIOM_DEPLOYMENT_URL")
		}
		if accessToken == "" {
			log.Fatal("missing AXIOM_ACCESS_TOKEN")
		}
	*/

	tcpParser := parser.New(func(msg *parser.Log) {
		fmt.Printf("tcp ")
		msg.PrettyPrint()
	})

	udpParser := parser.New(func(msg *parser.Log) {
		fmt.Printf("udp ")
		msg.PrettyPrint()
	})

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

	/*
		_, err = axiom.NewClient(deploymentURL, accessToken)
		if err != nil {
			log.Fatal(err)
		}
	*/

	time.Sleep(time.Second * 60)
}
