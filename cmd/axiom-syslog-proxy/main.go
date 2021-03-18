package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/axiomhq/axiom-syslog-proxy/input"
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

	closer, err := input.StartUDP(*addrUDP, onInput)
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close()

	closer, err = input.StartTCP(*addrTCP, onInput)
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

func onInput(line []byte, remoteIP string) {
	fmt.Println(string(line), "from", remoteIP)
}
