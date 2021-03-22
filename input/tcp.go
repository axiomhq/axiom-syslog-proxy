package input

import (
	"bufio"
	"io"
	"net"
	"time"
)

// TCPTimeout defines the maximum time between Reads
var TCPTimeout = time.Minute

// StartTCP ...
func StartTCP(addr string, cb WriteLineFunc) (io.Closer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	notifyCloser := NewNotifyCloser(listener)

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

			go handleTCPConnection(conn, cb)
		}
	}()

	return notifyCloser, nil
}

func handleTCPConnection(conn net.Conn, cb WriteLineFunc) {
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(TCPTimeout))

	host, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
	if splitErr != nil {
		host = conn.RemoteAddr().String()
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		_ = conn.SetReadDeadline(time.Now().Add(TCPTimeout))

		data := scanner.Bytes()
		cb(data, host)
	}

	err := scanner.Err()
	if err != nil {
		logger.Warn("Error reading TCP connection: %s (%s)", err, host)
	}
}
