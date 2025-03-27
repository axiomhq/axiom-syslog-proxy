package parser

import "maps"

func parseApp(msg *Log) {
	switch msg.Application {
	case "auth", "daemon", "kern", "syslog":
		if _, ok := msg.Metadata[logfileKey]; ok {
			parseSystemd(msg)
		}
	}
}

// systemd and auth don't come in with the header so we need to add it to parse them
func parseSystemd(msg *Log) {
	if m, _ := parseSyslogLine([]byte("<6> " + msg.Text)); m != nil {
		msg.Application = m.Application
		msg.Text = m.Text

		maps.Copy(msg.Metadata, m.Metadata)
	}
}
