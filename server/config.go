package server

const (
	fieldApplication = "application"
	fieldHostname    = "hostname"
	fieldSeverity    = "severity"
	fieldText        = "message"
	fieldMetadata    = "metadata"
	fieldRemoteAddr  = "remoteAddress"
)

// Config ...
type Config struct {
	Dataset string
	AddrUDP string
	AddrTCP string
}
