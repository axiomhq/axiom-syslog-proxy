package server

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/logmanager"

	"github.com/axiomhq/axiom-syslog-proxy/input"
	"github.com/axiomhq/axiom-syslog-proxy/parser"
)

var logger = logmanager.GetLogger("server.server")

const maxQueueSize = 1024

type Server struct {
	started   bool
	config    *Config
	client    *axiom.Client
	tcpCloser io.Closer
	udpCloser io.Closer
	tcpParser parser.Parser
	udpParser parser.Parser

	queue []axiom.Event
	mu    sync.RWMutex
}

func NewServer(client *axiom.Client, config *Config) (srv *Server, err error) {
	srv = &Server{
		config: config,
		client: client,
		queue:  make([]axiom.Event, 0, maxQueueSize),
	}

	srv.tcpParser = parser.New(srv.onLogMessage)
	srv.udpParser = parser.New(srv.onLogMessage)

	if srv.tcpCloser, err = input.StartTCP(config.AddrTCP, srv.tcpParser.WriteLine); err != nil {
		return nil, err
	}

	if srv.udpCloser, err = input.StartUDP(config.AddrUDP, srv.udpParser.WriteLine); err != nil {
		srv.tcpCloser.Close()
		return nil, err
	}

	return srv, nil
}

func (srv *Server) onLogMessage(log *parser.Log) {
	ev := LogToEvent(log)
	srv.mu.Lock()
	srv.queue = append(srv.queue, ev)
	needsFlushing := len(srv.queue) >= maxQueueSize
	srv.mu.Unlock()

	if needsFlushing {
		srv.Flush()
	}
}

func LogToEvent(log *parser.Log) axiom.Event {
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

func (srv *Server) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	srv.mu.Lock()
	defer srv.mu.Unlock()

	if len(srv.queue) == 0 {
		return nil
	}

	status, err := srv.client.Datasets.IngestEvents(ctx, srv.config.Dataset, axiom.IngestOptions{}, srv.queue...)
	if logger.IsError(err) {
		return err
	}

	logger.Trace("ingested %d event(s)", status.Ingested)
	srv.queue = make([]axiom.Event, 0, maxQueueSize)
	return nil
}

func (srv *Server) Run() {
	if srv.started {
		logger.Info("server already running")
		return
	}

	srv.started = true
	ticker := time.NewTicker(5 * time.Second)
	done := make(chan bool)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			srv.Flush()
		}
	}
}
