package syslog

import (
	"context"
	"io"
	"math"
	"math/rand"
	"sync"
	"time"

	axiomdb "axicode.axiom.co/watchmakers/axiomdb/client"
	commonAxDb "axicode.axiom.co/watchmakers/watchly/pkg/common/axiomdb"
	"axicode.axiom.co/watchmakers/watchly/pkg/common/system/types"
	"axicode.axiom.co/watchmakers/watchly/pkg/common/util"
	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/benchmarker"
	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/local"
	"axicode.axiom.co/watchmakers/watchly/pkg/uac/definitions"

	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/apis/syslog/input"
	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/apis/syslog/parser"
	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/metric"
	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/shared"

	"axicode.axiom.co/watchmakers/logmanager"
)

// maybe this should be configurable
const (
	queuedLogsDropThreshold = int(4 * 65536)
	maxQueuedLogs           = 2 * queuedLogsDropThreshold
)

var logger = logmanager.GetLogger("datastores.syslog")

type syslog struct {
	axDbClient *axiomdb.Client

	config *Config

	parser      parser.Parser
	linesMetric metric.Counter
	connsMetric metric.Gauge

	tcp  io.Closer
	tls  io.Closer
	udp  io.Closer
	unix io.Closer
	lock sync.Mutex

	buffer  *Queue
	monitor *queueMonitor

	stopChan  chan struct{}
	doneChan  chan struct{}
	flushChan chan struct{}

	indexedDocsMetric metric.Counter
	logsQLMetric      metric.Counter
	logsCountsMetric  metric.Counter
	logsStatsMetric   metric.Counter
	logsFacetsMetric  metric.Counter
	logsDroppedMetric metric.Counter

	*sync.RWMutex
}

var GetAxiomDBClient = func() (*axiomdb.Client, error) {
	ctx := local.UseLocalEntity(context.Background(), definitions.ServiceCore, []string{definitions.RoleManageSettings})
	if client, err := commonAxDb.GetLogsClient(ctx); err != nil {
		return nil, err
	} else {
		return client, nil
	}
}

// New creates a new Syslog implementation
func New(cfg *shared.Config) (shared.API, error) {
	InitVars()

	s := &syslog{}
	s.config = NewConfig()

	s.linesMetric = metric.NewCounter(metric.LogsProcessedLinesCounterName)
	s.connsMetric = metric.NewGauge(metric.SyslogTCPConnections)
	s.parser = parser.New(s.pipeLog)

	s.buffer = NewQueueWithMax(GetFlushThreshold(), maxQueuedLogs)
	s.monitor = newQueueMonitor()
	s.stopChan = make(chan struct{}, 1)
	s.flushChan = make(chan struct{}, 1)
	s.doneChan = make(chan struct{}, 1)

	s.indexedDocsMetric = metric.NewCounter(metric.LogsIndexedDocsCounterName)
	s.logsQLMetric = metric.NewCounter(metric.LogsQLQueries)
	s.logsCountsMetric = metric.NewCounter(metric.LogsCounts)
	s.logsStatsMetric = metric.NewCounter(metric.LogsStats)
	s.logsFacetsMetric = metric.NewCounter(metric.LogsFacets)
	s.logsDroppedMetric = metric.NewCounter(metric.LogsDroppedDocsCounterName)

	s.RWMutex = &sync.RWMutex{}

	return s, nil
}

func (s *syslog) Start(services []*shared.ServiceDetails) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if client, err := GetAxiomDBClient(); err != nil {
		return err
	} else {
		s.axDbClient = client
	}

	var err error

	for _, service := range services {
		if service.Name == types.ServiceNameSyslogTCP {
			s.config.TCPPort = service.Port
			s.config.TCPEnabled = service.Enabled
		}
		if service.Name == types.ServiceNameSyslogTLS {
			s.config.TLSPort = service.Port
			s.config.TLSEnabled = service.Enabled
		}
		if service.Name == types.ServiceNameSyslogUDP {
			s.config.UDPPort = service.Port
			s.config.UDPEnabled = service.Enabled
		}
	}

	if s.config.TCPEnabled {
		logger.Info("starting syslog tcp on %d ", s.config.TCPPort)
		if s.tcp, err = input.StartTCP(s.config.TCPPort, s.connsMetric, s.parser.WriteLine); err != nil {
			return err
		}
	} else {
		logger.Info("syslog tcp disabled")
	}

	if s.config.TLSEnabled {
		logger.Info("starting syslog tls on %d ", s.config.TLSPort)
		if s.tls, err = input.StartTLS(s.config.TLSPort, s.connsMetric, s.parser.WriteLine); err != nil {
			return err
		}
	} else {
		logger.Info("syslog tls disabled")
	}

	if s.config.UDPEnabled {
		logger.Info("starting syslog udp on %d ", s.config.UDPPort)
		if s.udp, err = input.StartUDP(s.config.UDPPort, s.parser.WriteLine); err != nil {
			return err
		}
	} else {
		logger.Info("syslog udp disabled")
	}

	if s.unix, err = input.StartUnix(s.parser.WriteLine); err != nil {
		return err
	}

	go s.flushloop()

	return nil
}

func (s *syslog) flushloop() {
	trimTicker := util.NewTicker(GetRetentionCheckTick())
	defer trimTicker.Stop()
	defer close(s.doneChan)

	for {
		select {
		case <-time.After(GetFlushTick()):
			logger.IsError(s.flush())
		case <-s.flushChan:
			logger.IsError(s.flush())
		case <-trimTicker.GetTicker():
			options := axiomdb.TrimOptions{
				MaxDuration: GetRetentionDays(),
				MaxSize:     math.MaxUint64,
			}
			if deletedBlocks, err := s.axDbClient.Datasets.Trim(context.Background(), commonAxDb.LogsDatasetName, options); err != nil {
				logger.Error("Error trimming logs DB: %s", err)
			} else if deletedBlocks.NumDeleted > 0 {
				logger.Info("Trimmed %d eventdb blocks", deletedBlocks)
			}
		case <-s.stopChan:
			logger.IsError(s.flush())
			return
		}
	}
}

// This is only triggered during shutdown, manual flush commands and every second
func (s *syslog) flush() error {
	start := time.Now()
	defer func() { _ = benchmarker.MeasureTime("syslog.flush.time", start, nil) }()
	s.Lock()
	defer s.Unlock()

	events := s.buffer.Get()
	if len(events) == 0 {
		return nil
	}

	if queueSize := s.buffer.size(); queueSize >= queuedLogsDropThreshold {
		// uh oh, trouble ahead
		if rate := s.monitor.GetProcessedRate(); rate > 1.0 {
			// rate = 1.0 -> drop 0%
			// rate = 2.0 -> drop 50%
			// rate = 4.0 -> drop 75%
			dropRate := 1 - (1 / rate)
			// drop a bit more, so there's a chance to actually clear the queue
			dropRate = math.Pow(dropRate, 0.85)
			droppedEvents := 0

			src := rand.NewSource(int64(queueSize))

			n := 0
			// filter the events slice in place
			for _, item := range events {
				// basically a src.Float64()
				rnd := float64(src.Int63()) / float64(1<<63)
				if rnd < dropRate {
					droppedEvents++
					continue
				}
				events[n] = item
				n++
			}
			s.logsDroppedMetric.Add(float64(droppedEvents))
			// dropped are like processed as far as the monitor is concerned
			s.monitor.AddProcessed(uint64(droppedEvents))
			events = events[:n]
			// FIXME: ideally we'd want the slice to contain common.MaxBlockNumRecs events
		} else if rate == 0 {
			// drop this entire batch, for some reason no logs are being processed
			droppedEvents := len(events)
			s.logsDroppedMetric.Add(float64(droppedEvents))
			// in this case we don't want to make the monitor think
			// these actually got processed, so not calling AddProcessed

			// and grab a new batch
			events = s.buffer.Get()
		}
	}

	var returnedError error

	if res, err := s.axDbClient.Datasets.IngestEvents(context.Background(), commonAxDb.LogsDatasetName, axiomdb.IngestOptions{}, events...); err != nil {
		logger.Error("Error sending logs to axiomdb %v", err)
		returnedError = err
	} else {
		batchLen := uint64(len(events))
		numFailures := res.Failed
		l := batchLen - numFailures
		s.indexedDocsMetric.Add(float64(l))
		s.monitor.AddProcessed(uint64(batchLen))

		for _, ingestFailure := range res.Failures {
			logger.Warn("Dropped %d (out of %d) events (%s)", numFailures, batchLen, ingestFailure.Error)
		}
	}

	return returnedError
}

func (s *syslog) Name() string {
	return "syslog"
}

func (s *syslog) Flush() error {
	return s.parser.Flush()
}

func (s *syslog) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.tcp != nil {
		logger.IsError(s.tcp.Close())
	}

	if s.tls != nil {
		logger.IsError(s.tls.Close())
	}

	if s.udp != nil {
		logger.IsError(s.udp.Close())
	}

	if s.unix != nil {
		logger.IsError(s.unix.Close())
	}

	s.parser.Stop()

	close(s.stopChan)
	<-s.doneChan
	close(s.flushChan)

	return nil
}

func (s *syslog) pipeLog(log *parser.Log) {
	s.linesMetric.Inc()
	evBatch := []map[string]interface{}{}
	output := map[string]interface{}{
		commonAxDb.DocFieldSeverity:    int64(log.Severity),
		commonAxDb.DocFieldHostname:    log.Hostname,
		commonAxDb.DocFieldApplication: log.Application,
		commonAxDb.DocFieldText:        log.Text,
		commonAxDb.DocFieldMetadata:    log.Metadata,
	}

	evBatch = append(evBatch, output)

	queueSize, dropped := s.buffer.Push(evBatch)
	if queueSize >= GetFlushThreshold() {
		// we only flush if there's not already a flush requested
		select {
		case s.flushChan <- struct{}{}:
		default:
		}
	}

	if dropped > 0 {
		s.logsDroppedMetric.Add(float64(dropped))
	}

	s.monitor.AddQueued(uint64(len(evBatch)))
}

func (s *syslog) EnsureEnabled(services []*shared.ServiceDetails) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var err error
	for _, service := range services {
		if service.Name == types.ServiceNameSyslogTCP {
			if !service.Enabled && s.config.TCPEnabled {
				if s.tcp != nil {
					logger.Info("close syslog tcp ...")
					logger.IsError(s.tcp.Close())
					s.tcp = nil
				}
				s.config.TCPEnabled = false
			} else if service.Enabled && !s.config.TCPEnabled {
				s.tcp, err = input.StartTCP(s.config.TCPPort, s.connsMetric, s.parser.WriteLine)
				if err != nil {
					logger.Warn("EnsureEnabled Syslog %v", err)
					return err
				}
				s.config.TCPEnabled = true
			}
		} else if service.Name == types.ServiceNameSyslogTLS {
			if !service.Enabled && s.config.TLSEnabled {
				if s.tls != nil {
					logger.Info("close syslog tls ...")
					logger.IsError(s.tls.Close())
					s.tls = nil
				}
				s.config.TLSEnabled = false
			} else if service.Enabled && !s.config.TLSEnabled {
				s.tls, err = input.StartTLS(s.config.TLSPort, s.connsMetric, s.parser.WriteLine)
				if err != nil {
					logger.Warn("EnsureEnabled Syslog %v", err)
					return err
				}
				s.config.TLSEnabled = true
			}
		} else if service.Name == types.ServiceNameSyslogUDP {
			if !service.Enabled && s.config.UDPEnabled {
				if s.udp != nil {
					logger.Info("close syslog udp ...")
					logger.IsError(s.udp.Close())
					s.udp = nil
				}
				s.config.UDPEnabled = false
			} else if service.Enabled && !s.config.UDPEnabled {
				s.udp, err = input.StartUDP(s.config.UDPPort, s.parser.WriteLine)
				if err != nil {
					logger.Warn("EnsureEnabled Syslog %v", err)
					return err
				}
				s.config.UDPEnabled = true
			}
		}
	}
	return nil
}
