package input

import (
	"bufio"
	"io"
	"net"

	"axicode.axiom.co/watchmakers/watchly/pkg/common/util"
)

func StartUnix(cb WriteLineFunc) (io.Closer, error) {
	if listener, err := util.UnixListen("service", "syslog-unix"); err != nil {
		return nil, err
	} else {
		notifyCloser := util.NewNotifyCloser(listener)

		go func() {
			logger.Info("Started unix server on %v:%v", listener.Addr().Network(), listener.Addr().String())
			for {
				conn, err := listener.Accept()
				if err != nil {
					if notifyCloser.WasClosed() {
						return
					}

					logger.IsError(err)

					continue
				}

				logger.Trace("New unix connection: %v", conn.RemoteAddr().String())
				go handleUnixConnection(conn, cb)
			}
		}()

		return notifyCloser, nil
	}
}

func handleUnixConnection(conn net.Conn, cb WriteLineFunc) {
	defer conn.Close()
	raddr := conn.RemoteAddr()
	scanner := bufio.NewScanner(conn)

	host, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
	if splitErr != nil {
		host = conn.RemoteAddr().String()
	}

	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) < 1 {
			if logger.IsDebugEnabled() {
				logger.Warn("Received message with no bytes from %s", raddr.String())
			}
			continue
		}
		cb(data, host)
	}

	err := scanner.Err()
	if err != nil {
		logger.Error("Error reading unix connection: %s (%s)", err, raddr.String())
	}
}
