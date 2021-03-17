package input

import (
	"axicode.axiom.co/watchmakers/watchly/pkg/common/sysinit"
	"axicode.axiom.co/watchmakers/watchly/pkg/common/util"
	"io"
	"net"
)

// StartUDP ...
func StartUDP(port int, cb WriteLineFunc) (io.Closer, error) {
	if port == 0 {
		logger.Warn("UDP server has been disabled")
		return nil, nil
	}

	if portFile, err := sysinit.RequestPort("udp", int32(port)); err != nil {
		return nil, err
	} else if conn, err := net.FilePacketConn(portFile); err != nil {
		return nil, logger.Error("Unable to start UDP server: %s", err)
	} else {
		notifyCloser := util.NewNotifyCloser(conn)

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
}

func getIP(ip string) string {
	if host, _, err := net.SplitHostPort(ip); err != nil {
		logger.Warn("Unable to split ip/port: %s", err)
	} else {
		return host
	}

	return ip
}
