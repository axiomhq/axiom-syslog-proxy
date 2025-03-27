package parser

import (
	fmt "fmt"
	"maps"
	"strings"
)

// Emergency...
const (
	Unknown   = -1
	Emergency = 0
	Alert     = 1
	Critical  = 2
	Error     = 3
	Warning   = 4
	Notice    = 5
	Info      = 6
	Debug     = 7
	Trace     = 8
)

// Severity ...
type Severity int

func (s Severity) String() string {
	switch s {
	case Emergency:
		return "Emergency"
	case Alert:
		return "Alert"
	case Critical:
		return "Critical"
	case Error:
		return "Error"
	case Warning:
		return "Warning"
	case Notice:
		return "Notice"
	case Info:
		return "Info"
	case Debug:
		return "Debug"
	case Trace:
		return "Trace"
	default:
		return "Unknown"
	}
}

// SeverityFromString parses Severity from a string
func SeverityFromString(s string) Severity {
	folded := strings.ToLower(s)

	switch folded {
	case "trace":
		return Trace
	case "debug":
		return Debug
	case "info":
		return Info
	case "notice":
		return Notice
	case "warning", "warn":
		return Warning
	case "error", "err":
		return Error
	case "critical", "crit":
		return Critical
	case "alert":
		return Alert
	case "emergency":
		return Emergency
	default:
		return Unknown
	}
}

// RawLogQuery is used by the rawlogquery aql function
type RawLogQuery struct {
	Limit      int
	Resolution string
	Filter     string
	GroupBy    []string
	Aggs       []string
	Order      []string
}

// Log ...
type Log struct {
	RemoteAddr  string
	Severity    int64
	Timestamp   int64
	Hostname    string
	Application string
	Text        string
	Metadata    map[string]any
}

func (l *Log) Merge(other *Log) {
	if other.Timestamp != 0 && other.Timestamp != l.Timestamp {
		l.Timestamp = other.Timestamp
	}
	if other.Severity != 0 && other.Severity != l.Severity {
		l.Severity = other.Severity
	}
	if other.RemoteAddr != "" {
		l.RemoteAddr = other.RemoteAddr
	}
	if other.Hostname != "" {
		l.Hostname = other.Hostname
	}
	if other.Application != "" {
		l.Application = other.Application
	}
	if other.Text != "" {
		l.Text = other.Text
	}
	maps.Copy(l.Metadata, other.Metadata)
}

// PrettyPrint ...
func (l *Log) PrettyPrint() {
	var metadata []string

	if l.Metadata != nil {
		for k, v := range l.Metadata {
			metadata = append(metadata, fmt.Sprintf("%s=%v", k, v))
		}
	}

	fmt.Printf(`
		RemoteAddr: %s
		Severity: %d
		Timestamp: %d
		Hostname: %s
		Application: %s
		Text: %s
		Metadata: %s
`, l.RemoteAddr, l.Severity, l.Timestamp, l.Hostname, l.Application, l.Text, strings.Join(metadata, ","))
}
