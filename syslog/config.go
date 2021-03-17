package syslog

// Config describes the configuration options for the syslog package
type Config struct {
	TCPPort int
	TLSPort int
	UDPPort int

	TCPEnabled bool
	TLSEnabled bool
	UDPEnabled bool
}

// NewConfig ...
func NewConfig() *Config {
	return &Config{
		TCPPort: 601,
		TLSPort: 6514,
		UDPPort: 514,
	}
}
