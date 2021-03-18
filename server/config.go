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
	URL     string
	Dataset string
	Token   string
	AddrUDP string
	AddrTCP string
}
