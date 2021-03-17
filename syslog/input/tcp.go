package input

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"time"

	"axicode.axiom.co/watchmakers/watchly/pkg/common/certs"
	"axicode.axiom.co/watchmakers/watchly/pkg/common/sysinit"
	"axicode.axiom.co/watchmakers/watchly/pkg/common/util"
	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/metric"
)

// TCPTimeout defines the maximum time between Reads
var TCPTimeout = time.Minute

// StartTCP ...
func StartTCP(port int, connsMetric metric.Gauge, cb WriteLineFunc) (io.Closer, error) {
	if port == 0 {
		logger.Warn("TCP server has been disabled")
		return nil, nil
	}

	if portFile, err := sysinit.RequestPort("tcp", int32(port)); err != nil {
		return nil, err
	} else if listener, err := net.FileListener(portFile); err != nil {
		return nil, logger.Error("Unable to start TCP server: %s", err)
	} else {
		notifyCloser := util.NewNotifyCloser(listener)

		go func() {
			logger.Info("Started TCP server on %v:%v", listener.Addr().Network(), listener.Addr().String())
			for {
				conn, err := listener.Accept()
				if err != nil {
					if notifyCloser.WasClosed() {
						return
					}

					logger.IsError(err)

					continue
				}

				util.GlobalLimiter.Test(conn, func() {
					logger.Trace("New TCP connection: %v", conn.RemoteAddr().String())
					go handleTCPConnection(conn, connsMetric, cb)
				})
			}
		}()

		return notifyCloser, nil
	}
}

// StartTLS ...
func StartTLS(port int, connsMetric metric.Gauge, cb WriteLineFunc) (io.Closer, error) {
	if port == 0 {
		logger.Warn("TLS server has been disabled")
		return nil, nil
	}

	crts := certs.NewReadOnly()
	cfg := &tls.Config{GetCertificate: crts.GetOnHello}

	if portFile, err := sysinit.RequestPort("tcp", int32(port)); err != nil {
		return nil, err
	} else if fileListener, err := net.FileListener(portFile); err != nil {
		return nil, logger.Error("Unable to start TLS server: %s", err)
	} else {
		listener := tls.NewListener(fileListener, cfg)
		notifyCloser := util.NewNotifyCloser(listener)

		go func() {
			logger.Info("Started TLS server on %v:%v", listener.Addr().Network(), listener.Addr().String())

			for {
				conn, err := listener.Accept()
				if err != nil {
					if notifyCloser.WasClosed() {
						return
					}

					logger.IsError(err)

					continue
				}

				util.GlobalLimiter.Test(conn, func() {
					logger.Trace("New TLS connection: %v", conn.RemoteAddr().String())
					go handleTCPConnection(conn, connsMetric, cb)
				})
			}
		}()
		return notifyCloser, nil
	}
}

func handleTCPConnection(conn net.Conn, connsMetric metric.Gauge, cb WriteLineFunc) {
	defer util.GlobalLimiter.Close(conn)
	if connsMetric != nil {
		connsMetric.Inc()
		defer connsMetric.Dec()
	}
	conn.SetReadDeadline(time.Now().Add(TCPTimeout))

	host, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
	if splitErr != nil {
		host = conn.RemoteAddr().String()
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		conn.SetReadDeadline(time.Now().Add(TCPTimeout))

		data := scanner.Bytes()
		cb(data, host)
	}

	err := scanner.Err()
	if err != nil {
		logger.Warn("Error reading TCP connection: %s (%s)", err, host)
	}
}
