package parser

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

const (
	logfileKey   = "axiom.logfile"
	maxNestLevel = 5
)

var (
	timestampKeys = map[string]bool{"syslog.timestamp": true, "timestamp": true, "eventtime": true, "@timestamp": true, "_timestamp": true, "date": true, "published_date": true}
	hostKeys      = map[string]bool{"syslog.hostname": true, "hostname": true, "host": true}
	appKeys       = map[string]bool{"syslog.appname": true, "app": true, "application": true}
	msgKeys       = map[string]bool{"message": true, "msg": true}
	severityKeys  = map[string]bool{"syslog.severity": true, "severity": true, "status": true, "level": true}
	jsonKeysRegex = regexp.MustCompile(`[\.\[\]]`)
)

// ParseLineWithFallback parses an individual line, and creates a message if the line is not valid
func ParseLineWithFallback(line []byte, remoteAddr string) *Log {
	var m *Log
	var err error

	if ok, jsonMsg := detectMaybeJSON(line); ok {
		m, err = parseJSON(jsonMsg)
		// if the message is not valid json, fallback to syslog
		if err != nil {
			log.Printf("Unable to parse log line, err=%q: %s", err, line)
			m, err = parseSyslogLine(line)
		}
	} else {
		m, err = parseSyslogLine(line)
	}

	if err != nil {
		log.Printf("Unable to parse log line: %s", line)

		if err == errCorruptedData {
			return nil
		}
		if len(line) < 1 {
			return nil
		}
		if m, err = syntheticLog(remoteAddr, line); err != nil {
			return nil
		}
	}

	m.RemoteAddr = remoteAddr

	if m.Hostname == "" {
		m.Hostname = remoteAddr
	}

	if m.Timestamp == 0 {
		m.Timestamp = time.Now().UnixNano()
	}

	parseApp(m)

	// attempt to parse json from the text property
	if ok, msg := detectMaybeJSON([]byte(m.Text)); ok {
		sublog, err := parseJSON(msg)
		if err == nil {
			// merge the sublog with the main log
			m.Merge(sublog)
		}
	}

	// Always last
	populateSeverity(m)

	return m
}

// detectMaybeJSON finds hints of a json message but does not
// guarantee that the message is actually json parsable.
func detectMaybeJSON(line []byte) (ok bool, result []byte) {
	if len(line) < 1 {
		return
	}

	// search backwards for } as json messages are always going to end
	// with }
	var farIndex int
	for i := len(line) - 1; i >= 0; i-- {
		// Note: not unicode safe, but will always fail if unicode characters are present
		switch line[i] {
		case ' ':
			continue
		case '}':
			farIndex = i
		default:
			return
		}
		break
	}

	// sanity check
	if farIndex < 1 {
		return
	}

	// look for either a syslog PRE (https://datatracker.ietf.org/doc/html/rfc5424#section-6)
	// or a json message
	// we only support looking for a PRE as a special case to avoid RFC5424 parsing
	// json fields as appName/Hostname
	for i := 0; i < len(line); i++ {
		// Note: not unicode safe, but syslog does not allow unicode characters
		// so will fail before unicode becomes an issue
		switch line[i] {
		case ' ':
			continue
		case '<':
			// PRE start, zoom forward to the end
			for ; i < len(line); i++ {
				switch line[i] {
				case '>':
					break
				default:
					continue
				}
				break
			}
		case '{':
			return true, line[i : farIndex+1]
		default:
			return
		}
	}
	return
}

// parseJSON takes a single json message to parse
func parseJSON(data []byte) (*Log, error) {
	msg := &Log{
		Metadata: map[string]interface{}{},
	}

	if err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, _ int) error {
		return extractJSONProperty(key, value, dataType, msg)
	}); err != nil {
		return nil, err
	}

	return msg, nil
}

func extractJSONProperty(key []byte, value []byte, dataType jsonparser.ValueType, msg *Log) error {
	loweredString := strings.ToLower(string(key))
	if dataType == jsonparser.String {
		stringValue, parseErr := jsonparser.ParseString(value)
		if parseErr != nil {
			return parseErr
		}
		if timestampKeys[loweredString] {
			if t, e := time.Parse(time.RFC3339Nano, stringValue); e == nil {
				msg.Timestamp = t.UnixNano()
			} else {
				msg.Metadata["unparsed_timestamp"] = stringValue
			}
		} else if hostKeys[loweredString] {
			msg.Hostname = stringValue
		} else if appKeys[loweredString] {
			msg.Application = stringValue
		} else if msgKeys[loweredString] {
			msg.Text = stringValue
		} else if severityKeys[loweredString] {
			msg.Severity = int64(SeverityFromString(string(value)))
		} else {
			if err := extractMetadataValue(joinKey("", string(key)), value, dataType, 0, msg); err != nil {
				return err
			}
		}
	} else {
		if err := extractMetadataValue(joinKey("", string(key)), value, dataType, 0, msg); err != nil {
			return err
		}
	}
	return nil
}

func extractMetadataValue(concatKey string, value []byte, dataType jsonparser.ValueType, level int64, msg *Log) error {
	if level > maxNestLevel {
		return nil
	}

	switch dataType {
	case jsonparser.Object:
		level++
		if err := jsonparser.ObjectEach(value, func(kk []byte, vv []byte, dtdt jsonparser.ValueType, _ int) error {
			return extractMetadataValue(joinKey(concatKey, string(kk)), vv, dtdt, level, msg)
		}); err != nil {
			return err
		}
	case jsonparser.Array:
		arrayIndex := 0
		level++
		if _, err := jsonparser.ArrayEach(value, func(vv []byte, dtdt jsonparser.ValueType, _ int, err error) {
			if err != nil {
				return
			}
			newConcatKey := fmt.Sprintf("%s[%d]", concatKey, arrayIndex)
			if err := extractMetadataValue(newConcatKey, vv, dtdt, level, msg); err != nil {
				return
			}
			arrayIndex++
		}); err != nil {
			return err
		}
	case jsonparser.Number:
		if n, err := ParseInt(value); err == nil {
			msg.Metadata[concatKey] = n
		} else if f, err := ParseFloat(value); err == nil {
			msg.Metadata[concatKey] = f
		}
	case jsonparser.Boolean:
		msg.Metadata[concatKey] = string(value)
	case jsonparser.String:
		stringValue, parseErr := jsonparser.ParseString(value)
		if parseErr != nil {
			return parseErr
		}
		msg.Metadata[concatKey] = stringValue
	case jsonparser.NotExist, jsonparser.Null, jsonparser.Unknown:
		fallthrough
	default:
		log.Printf("JSON type %v is unsupported", dataType)
	}
	return nil
}

func joinKey(parent string, child string) string {
	if jsonKeysRegex.MatchString(child) {
		child = fmt.Sprintf("\"%s\"", child)
	}
	if len(parent) == 0 {
		return child
	}
	return fmt.Sprintf("%s.%s", parent, child)
}

// parseSyslogLine takes a single syslog message to parse
func parseSyslogLine(data []byte) (*Log, error) {
	if bytes.IndexByte(data, '<') != 0 {
		return nil, errParse
	}
	return parseSyslog(data)
}

func syntheticLog(host string, msg []byte) (*Log, error) {
	line := fmt.Sprintf("<14>%s %s %s: %s", time.Now().UTC().Format(time.RFC3339), host, "unknown", bytes.TrimSpace(msg))
	return parseSyslogLine([]byte(line))
}

func extractSeverity(text string) int32 {
	length := len(text)

	test := func(i int, s string, slen int) bool {
		if i+slen > length {
			return false
		}

		for j := 0; j < slen; j++ {
			c := text[i+j]
			n := s[j]
			if c != n && c != 'A'+n-'a' {
				return false
			}
		}
		return true
	}

	for i, c := range text {
		switch c {
		case 'c':
			if test(i+1, "rit", 3) {
				return Critical
			}
		case 'C':
			if test(i+1, "RIT", 3) {
				return Critical
			}
		case 'e':
			if test(i+1, "rror", 4) {
				return Error
			}
		case 'E':
			if test(i+1, "RR", 2) {
				return Error
			}
		case 'w':
			if test(i+1, "arn", 3) {
				return Warning
			}
		case 'W':
			if test(i+1, "ARN", 3) {
				return Warning
			}
		case 'i':
			if test(i+1, "nfo", 3) {
				return Info
			}
		case 'I':
			if test(i+1, "NFO", 3) {
				return Info
			}
		case 'd':
			if test(i+1, "ebug", 4) {
				return Debug
			}
		case 'D':
			if test(i+1, "EBUG", 4) {
				return Debug
			}
		case 't':
			if test(i+1, "race", 4) {
				return Trace
			}
		case 'T':
			if test(i+1, "RACE", 4) {
				return Trace
			}
		}
	}

	return Unknown
}

// After calling this, the only severities a message will have are:
// Error, Warning, Info, Debug, or Trace
// These are the ones that have corressponding UX stuff in the dashboard (colours etc)
func populateSeverity(msg *Log) {
	if msg.Severity == Unknown {
		msg.Severity = Info
	}

	if msg.Severity == Notice {
		msg.Severity = Info
	}

	if msg.Severity < Error {
		msg.Severity = Error
	}

	// Override with what's in the text
	if severity := extractSeverity(msg.Text); severity != Unknown {
		msg.Severity = int64(severity)
	}
}
