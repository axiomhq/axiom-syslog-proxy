package input

import (
	"io"
	"net"
)

// StartUDP ...
func StartUDP(addr string, cb WriteLineFunc) (io.Closer, error) {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}

	notifyCloser := NewNotifyCloser(conn)

	go func() {
		logger.Info("Started UDP server on %v:%v", conn.LocalAddr().Network(), conn.LocalAddr().String())

		for {
			var jumboPacket [8960]byte
			bytesRead, raddr, packetErr := conn.ReadFrom(jumboPacket[:])

			if packetErr == io.EOF {
				return
			} else if packetErr != nil {
				if notifyCloser.WasClosed() {
					return
				}

				logger.IsError(packetErr)

				continue
			} else if bytesRead < 1 {
				logger.Warn("Received message with no bytes from %v", raddr.String())
				continue
			}

			cb(jumboPacket[:bytesRead], getIP(raddr.String()))
		}
	}()

	return notifyCloser, nil
}

func getIP(ip string) string {
	if host, _, err := net.SplitHostPort(ip); err != nil {
		logger.Warn("Unable to split ip/port: %s", err)
	} else {
		return host
	}

	return ip
}
