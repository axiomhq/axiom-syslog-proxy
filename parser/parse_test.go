package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ParseTestSuite struct {
	suite.Suite
}

func TestParseTestSuite(t *testing.T) {
	suite.Run(t, new(ParseTestSuite))
}

type Case struct {
	raw           []byte
	time          time.Time
	hostname      string
	application   string
	text          string
	metadata      map[string]interface{}
	metadataLTLen int
	imprecise     bool
	severity      int64
	adjust        bool
}

func (s *ParseTestSuite) TestParseJson() {
	require := s.Require()
	now := time.Now()
	nowFormatted := now.Format(time.RFC3339Nano)

	cases := []*Case{
		{
			raw:         []byte(fmt.Sprintf(`{"severity":"info", "data" : [0,"one",{"number":"deux"}, 3.3, false], "annoy[ing]": "value", "artist": "Tomonari Nozaki", "album": "North Palace", "message": "Favourite album", "application":"logstash", "hostname":"forwind.net", "timestamp": "%s"}`, nowFormatted)),
			application: "logstash",
			time:        now,
			hostname:    "forwind.net",
			text:        "Favourite album",
			severity:    int64(Info),
			metadata:    map[string]interface{}{"\"annoy[ing]\"": "value", "artist": "Tomonari Nozaki", "album": "North Palace", "data[0]": int64(0), "data[1]": "one", "data[2].number": "deux", "data[3]": 3.3, "data[4]": "false"},
		},
		{
			raw:         []byte(fmt.Sprintf(`{"syslog.severity":"info", "oh.no": ":(", "oh": {"no[7]": ":((("}, "artist": "Fourth Page", "album": "Along the weak rope", "Msg": "Least Favourite album", "app":"logstash", "host":"forwind.net", "Timestamp": "%s"}`, nowFormatted)),
			application: "logstash",
			time:        now,
			hostname:    "forwind.net",
			text:        "Least Favourite album",
			severity:    int64(Info),
			metadata:    map[string]interface{}{"artist": "Fourth Page", "album": "Along the weak rope", "oh.\"no[7]\"": ":(((", "\"oh.no\"": ":("},
		},
		{
			raw:         []byte(fmt.Sprintf(`{"level":"debug", "msg": "Best recent 1", "a.h[a]": {"ta.ke" : ["on", "m.e"], "float": 4.3, "bo[ol]" : false}, "artist": "Rune Clausen", "album": "Tones Jul", "application":"logstash", "syslog.hostname":"forwind.net", "syslog.timestamp":"%s"}`, nowFormatted)),
			time:        now,
			application: "logstash",
			hostname:    "forwind.net",
			text:        "Best recent 1",
			severity:    int64(Debug),
			metadata:    map[string]interface{}{"artist": "Rune Clausen", "album": "Tones Jul", "\"a.h[a]\".\"ta.ke\"[0]": "on", "\"a.h[a]\".\"ta.ke\"[1]": "m.e", "\"a.h[a]\".float": 4.3, "\"a.h[a]\".\"bo[ol]\"": "false"},
		},
		{
			raw:         []byte(fmt.Sprintf(`{"level":"trace", "msg": "Best recent 2", "artist": "Rune Clausen", "album": "Tones Jul", "application":"logstash", "syslog.hostname":"forwind.net", "syslog.timestamp":"%s"}`, nowFormatted)),
			time:        now,
			application: "logstash",
			hostname:    "forwind.net",
			text:        "Best recent 2",
			severity:    int64(Trace),
			metadata:    map[string]interface{}{"artist": "Rune Clausen", "album": "Tones Jul"},
		},
		{
			raw:         []byte(fmt.Sprintf(`{"level":"trace", "msg": "Best recent 3", "bool": true, "forwind": {"favourites":  {"artist" : "Rune Clausen", "album": "Blindlight", "release" : { "duration" : 100, "catno" : "fwd09", "link" : { "url" : "http://www.forwind.net", "type" : {"origin": "home", "ignore": {"this": "we shouldn't parse this"}}}}}}, "application":"logstash", "syslog.hostname":"forwind.net", "syslog.timestamp":"%s"}`, nowFormatted)),
			time:        now,
			application: "logstash",
			hostname:    "forwind.net",
			text:        "Best recent 3",
			severity:    int64(Trace),
			metadata:    map[string]interface{}{"forwind.favourites.artist": "Rune Clausen", "bool": "true", "forwind.favourites.release.link.type.origin": "home", "forwind.favourites.album": "Blindlight", "forwind.favourites.release.duration": int64(100), "forwind.favourites.release.catno": "fwd09", "forwind.favourites.release.link.url": "http://www.forwind.net"},
		},
	}

	for number, c := range cases {
		msg := ParseLineWithFallback(c.raw, "forwind.net")
		str := fmt.Sprintf("%d  %s", number, string(c.raw))
		require.NotNil(msg)

		if c.hostname != "" {
			require.Equal(c.hostname, msg.Hostname, str)
		}
		require.Equal(c.application, msg.Application, str)
		require.Equal(c.text, msg.Text, str)
		require.Equal(c.severity, msg.Severity, str)

		if c.time.Year() != 0 {
			ts := time.Unix(0, msg.Timestamp).In(c.time.Location())
			require.Equal(c.time.Year(), ts.Year(), str)
			require.Equal(c.time.Month(), ts.Month(), str)
			require.Equal(c.time.Day(), ts.Day(), str)
			require.Equal(c.time.Hour(), ts.Hour(), str)
			require.Equal(c.time.Minute(), ts.Minute(), str)
		}

		for k, v := range c.metadata {
			require.Equal(v, msg.Metadata[k], fmt.Sprintf("%d metadata mismatch on key '%s': %+v\n", number, k, msg.Metadata))
		}
		assert := assert.New(s.T())
		assert.Len(msg.Metadata, len(c.metadata))
	}
}

func (s *ParseTestSuite) TestParse() {
	require := s.Require()
	now := time.Now()

	cases := []*Case{
		/* RFC3164: priority, date, and time */
		{
			raw:         []byte("<15> openvpn[2499]: PTHREAD support initialized"),
			time:        now,
			application: "openvpn",
			text:        "PTHREAD support initialized",
			imprecise:   true,
		},
		{
			raw:         []byte("<15> redis: \xef\xbb\xbfutf8isbom"),
			time:        now,
			application: "redis",
			text:        "utf8isbom",
			imprecise:   true,
		},
		{
			raw:         []byte("<15> openvpn[2499]: PTHREAD support initialized"),
			time:        now,
			application: "openvpn",
			text:        "PTHREAD support initialized",
			imprecise:   true,
		},
		{
			raw:         []byte(`<14> src time="2018-06-02T17:16:14.392415523+01:00" bool=false level=info float=5.6 number=3 msg="[graphdriver] using prior storage driver: aufs"`),
			time:        now,
			application: "src",
			text:        `time="2018-06-02T17:16:14.392415523+01:00" bool=false level=info float=5.6 number=3 msg="[graphdriver] using prior storage driver: aufs"`,
			imprecise:   true,
			metadata:    map[string]interface{}{"time": "2018-06-02T17:16:14.392415523+01:00", "bool": "false", "level": "info", "float": float64(5.6), "number": int64(3)},
		},
		{
			raw:         []byte("<15>Jan  1 01:00:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(now.Year(), 1, 1, 1, 0, 0, 0, time.Local),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<15>Jan 10 01:00:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(now.Year(), 1, 10, 1, 0, 0, 0, time.Local),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<13>Jan  1 14:40:51 alma korte: message"),
			time:        time.Date(now.Year(), 1, 1, 14, 40, 51, 0, time.Local),
			hostname:    "alma",
			application: "korte",
			text:        "message",
		},
		{
			raw:         []byte("<7>2006-11-10T10:43:21.156+02:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 11, 10, 8, 43, 21, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-11-10T10:43:21.156+01:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 11, 10, 9, 43, 21, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-03-26T01:59:59.156+01:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 03, 26, 0, 59, 59, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-03-26T02:00:00.156+01:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 03, 26, 1, 0, 0, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T01:00:00.156+02:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 28, 23, 0, 0, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T01:59:59.156+02:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 28, 23, 59, 59, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T02:00:00.156+02:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 00, 00, 00, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T02:00:00.15+02:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 00, 00, 00, 150000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T01:00:00.156+01:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 00, 00, 00, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T01:59:59.156+01:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 00, 59, 59, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T02:00:00.156+01:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 1, 00, 00, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		// RFC3164: hostname
		{
			raw:         []byte("<7>2006-10-29T02:00:00.156+01:00 %bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 1, 00, 00, 156000000, time.UTC),
			hostname:    "%bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T02:00:00.156+01:00 bzorp openvpn[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 1, 00, 00, 156000000, time.UTC),
			hostname:    "bzorp",
			application: "openvpn",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>2006-10-29T02:00:00.156+01:00 "),
			time:        time.Date(2006, 10, 29, 1, 00, 00, 156000000, time.UTC),
			hostname:    "",
			application: "",
			text:        "",
		},
		{
			raw:         []byte("<7>2006-10-29T02:00:00.156+01:00"),
			time:        time.Date(2006, 10, 29, 1, 00, 00, 156000000, time.UTC),
			hostname:    "",
			application: "",
			text:        "",
		},
		{
			raw:         []byte("<7>2006-10-29T02:00:00.156+01:00 ctld snmpd[2499]: PTHREAD support initialized"),
			time:        time.Date(2006, 10, 29, 1, 00, 00, 156000000, time.UTC),
			hostname:    "ctld",
			application: "snmpd",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7> Aug 29 02:00:00.156 ctld snmpd[2499]: PTHREAD support initialized"),
			time:        time.Date(now.Year(), 8, 29, 2, 00, 00, 156000000, time.Local),
			hostname:    "ctld",
			application: "snmpd",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7> Aug 29 02:00:00. ctld snmpd[2499]: PTHREAD support initialized"),
			time:        time.Date(now.Year(), 8, 29, 2, 00, 00, 0, time.Local),
			hostname:    "ctld",
			application: "snmpd",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7> Aug 29 02:00:00 ctld snmpd[2499]: PTHREAD support initialized"),
			time:        time.Date(now.Year(), 8, 29, 2, 00, 00, 0, time.Local),
			hostname:    "ctld",
			application: "snmpd",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<7>Aug 29 02:00:00 bzorp ctld/snmpd[2499]: PTHREAD support initialized"),
			time:        time.Date(now.Year(), 8, 29, 2, 00, 00, 0, time.Local),
			hostname:    "bzorp",
			application: "ctld/snmpd",
			text:        "PTHREAD support initialized",
		},
		{
			raw:         []byte("<190>Apr 15 2007 21:28:13: %PIX-6-302014: Teardown TCP connection 1688438 for bloomberg-net:1.2.3.4/8294 to inside:5.6.7.8/3639 duration 0:07:01 bytes 16975 TCP FINs"),
			time:        time.Date(2007, 4, 15, 21, 28, 13, 0, time.Local),
			hostname:    "",
			application: "%PIX-6-302014",
			text:        "Teardown TCP connection 1688438 for bloomberg-net:1.2.3.4/8294 to inside:5.6.7.8/3639 duration 0:07:01 bytes 16975 TCP FINs",
		},
		{
			raw:         []byte("<190>Apr 15 2007 21:28:13 %ASA: this is a Cisco ASA timestamp"),
			time:        time.Date(2007, 4, 15, 21, 28, 13, 0, time.Local),
			application: "%ASA",
			text:        "this is a Cisco ASA timestamp",
		},
		{
			raw:         []byte("<38>Sep 22 10:11:56 cdaix66 sshd[679960]: Accepted publickey for nagios from 1.9.1.1 port 42096 ssh2"),
			time:        time.Date(now.Year(), 9, 22, 10, 11, 56, 0, time.Local),
			hostname:    "cdaix66",
			application: "sshd",
			text:        "Accepted publickey for nagios from 1.9.1.1 port 42096 ssh2",
		},
		{
			raw:         []byte("<38>Apr  8 10:03:21 XPS-13-9380 gnome-shell[2332]: Error invoking IBus.set_global_engine_async: Expected function for callback argument callback, got undefined#012setEngine@resource:///org/gnome/shell/misc/ibusManager.js:207:9#012wrapper@resource:///org/gnome/gjs/modules/_legacy.js:82:22"),
			time:        time.Date(now.Year(), 4, 8, 10, 3, 21, 0, time.Local),
			hostname:    "XPS-13-9380",
			application: "gnome-shell",
			text:        "Error invoking IBus.set_global_engine_async: Expected function for callback argument callback, got undefined\nsetEngine@resource:///org/gnome/shell/misc/ibusManager.js:207:9\nwrapper@resource:///org/gnome/gjs/modules/_legacy.js:82:22",
		},
		{
			raw:         []byte("Use the BFG!"),
			time:        now,
			application: "unknown",
			text:        "Use the BFG!",
			imprecise:   true,
		},

		// RFC5424
		{
			raw:         []byte("<7>1 2006-10-29T01:59:59.156+01:00 mymachine.example.com evntslog - ID47 [exampleSDID@0 iut=\"3\" eventSource=\"Application\" eventID=\"1011\"][examplePriority@0 class=\"high\"] \xEF\xBB\xBF An application event log entry..."),
			time:        time.Date(2006, 10, 29, 0, 59, 59, 156000000, time.UTC),
			hostname:    "mymachine.example.com",
			application: "evntslog",
			text:        "An application event log entry...",
			metadata:    map[string]interface{}{"exampleSDID.iut": "3", "examplePriority.class": "high", "exampleSDID.eventID": "1011", "exampleSDID.eventSource": "Application"},
		},
		{
			raw:         []byte("<6>1 2018-08-09T07:19:28.698693Z mymachine.example.com evntslog - ID47 - \xEF\xBB\xBFAn application event log entry..."),
			time:        time.Date(2018, 8, 9, 7, 19, 28, 698693000, time.UTC),
			hostname:    "mymachine.example.com",
			application: "evntslog",
			text:        "An application event log entry...",
			metadata:    map[string]interface{}{},
		},
		{
			raw:         []byte("<7>1 2006-10-29T01:59:59.156Z mymachine.example.com evntslog - ID47 [exampleSDID@0 iut=\"3\" eventSource=\"Application\" eventID=\"1011\"][examplePriority@0 class=\"high\"] \xEF\xBB\xBF An application event log entry..."),
			time:        time.Date(2006, 10, 29, 1, 59, 59, 156000000, time.UTC),
			hostname:    "mymachine.example.com",
			application: "evntslog",
			text:        "An application event log entry...",
			metadata:    map[string]interface{}{"exampleSDID.iut": "3", "examplePriority.class": "high", "exampleSDID.eventID": "1011", "exampleSDID.eventSource": "Application"},
		},
		{
			raw:         []byte("<7>1 2006-10-29T01:59:59.156Z mymachine.example.com evntslog - ID47 [ exampleSDID@0 iut=\"3\" eventSource=\"App\\\"lication\\]\" eventID=\"1011\"][examplePriority@0 class=\"high_class\"] \xEF\xBB\xBF An application event log entry..."),
			time:        time.Date(2006, 10, 29, 1, 59, 59, 156000000, time.UTC),
			hostname:    "mymachine.example.com",
			application: "evntslog",
			text:        "An application event log entry...",
			metadata:    map[string]interface{}{"exampleSDID.iut": "3", "examplePriority.class": "high_class", "exampleSDID.eventID": "1011", "exampleSDID.eventSource": "App\"lication]"},
		},
		{
			raw:           []byte("<7>1 2006-10-29T01:59:59.156Z mymachine.example.com evntslog - ID47 - Running executor with --project=axiom .env=development"),
			time:          time.Date(2006, 10, 29, 1, 59, 59, 156000000, time.UTC),
			hostname:      "mymachine.example.com",
			application:   "evntslog",
			text:          "Running executor with --project=axiom .env=development",
			metadataLTLen: 1,
		},
		{
			raw:           []byte("<14>2018-06-19T11:08:00-07:00 bar elasticsearch: [2018-06-19 11:08:00,000][DEBUG][gateway] [Blizzard II] recovered [0] indices into cluster_state"),
			time:          time.Date(2018, 6, 19, 18, 8, 00, 0, time.UTC),
			hostname:      "bar",
			application:   "elasticsearch",
			text:          "[2018-06-19 11:08:00,000][DEBUG][gateway] [Blizzard II] recovered [0] indices into cluster_state",
			metadataLTLen: 0,
		},
		{
			raw:           []byte("<34>1 1987-01-01T12:00:27.156+00:20 192.0.2.1 myproc 8710 - - %% It's time to make the do-nuts.="),
			time:          time.Date(1987, 1, 1, 11, 40, 27, 156000000, time.UTC),
			hostname:      "192.0.2.1",
			application:   "myproc",
			text:          "%% It's time to make the do-nuts.=",
			metadataLTLen: 0,
		},
		{
			raw:           []byte("<134>1 2009-10-16T11:51:56+02:00 exchange.macartney.esbjerg MSExchange_ADAccess 20208 - - = hello"),
			time:          time.Date(2009, 10, 16, 9, 51, 56, 0, time.UTC),
			hostname:      "exchange.macartney.esbjerg",
			application:   "MSExchange_ADAccess",
			text:          "= hello",
			metadataLTLen: 1,
		},
		{
			raw:         []byte("<134>1 2009-10-16T11:51:56+02:00 2001:0db8:85a3:0000:0000:8a2e:0370:7334 MSExchange_ADAccess 20208 - - hello customer=njpatel@gmail.com source=web plan=\"professional plus\" foo= =bar hi"),
			time:        time.Date(2009, 10, 16, 9, 51, 56, 0, time.UTC),
			hostname:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			application: "MSExchange_ADAccess",
			text:        "hello customer=njpatel@gmail.com source=web plan=\"professional plus\" foo= =bar hi",
			metadata:    map[string]interface{}{"customer": "njpatel@gmail.com", "source": "web", "plan": "professional plus"},
		},
		{
			raw:         []byte("<134>1 2009-10-16T11:51:56+02:00 www web - - - \"customer id\"=\"njpatel@gmail.com\" \"source_app\"=web plan=\"professional plus\" foo= =bar = \"region\"="),
			time:        time.Date(2009, 10, 16, 9, 51, 56, 0, time.UTC),
			hostname:    "www",
			application: "web",
			text:        "\"customer id\"=\"njpatel@gmail.com\" \"source_app\"=web plan=\"professional plus\" foo= =bar = \"region\"=",
			metadata:    map[string]interface{}{"customer id": "njpatel@gmail.com", "source_app": "web", "plan": "professional plus"},
		},
		{
			raw:         []byte("<134>1 2009-10-16T11:51:56+02:00 www web - - - customer=\"njpatel@gmail.com\" \"source_app\"=web plan=\"professional plus\" foo= =bar = =\"region\""),
			time:        time.Date(2009, 10, 16, 9, 51, 56, 0, time.UTC),
			hostname:    "www",
			application: "web",
			text:        "customer=\"njpatel@gmail.com\" \"source_app\"=web plan=\"professional plus\" foo= =bar = =\"region\"",
			metadata:    map[string]interface{}{"customer": "njpatel@gmail.com", "source_app": "web", "plan": "professional plus"},
		},
		{
			raw:           []byte("<134>1 2009-10-16T11:51:56+02:00 www dash - - - GET 403 /api/v1/logs?groups=&last-log=2018-06-22T15%3A21%3A47.085654-07%3A00&delta=100 localhost:8080 ip=::1"),
			time:          time.Date(2009, 10, 16, 9, 51, 56, 0, time.UTC),
			hostname:      "www",
			application:   "dash",
			text:          "GET 403 /api/v1/logs?groups=&last-log=2018-06-22T15%3A21%3A47.085654-07%3A00&delta=100 localhost:8080 ip=::1",
			metadata:      map[string]interface{}{"ip": "::1"},
			metadataLTLen: 2,
		},
		{
			raw:         []byte("<6>1 2018-06-04T16:43:18.874822+01:00 XPS-15-9560 kernel - - - device lo entered promiscuous mode"),
			time:        time.Date(2018, 6, 4, 15, 43, 18, 874822000, time.UTC),
			hostname:    "XPS-15-9560",
			application: "kernel",
			text:        "device lo entered promiscuous mode",
		},
		{
			raw:         []byte("<6>1 2018-06-04T16:43:18.874822+01:00 XPS-15-9560 org.gnome.Shell.desktop 2136 - - == Stack trace for context 0x563cea7c7340 =="),
			time:        time.Date(2018, 6, 4, 15, 43, 18, 874822000, time.UTC),
			hostname:    "XPS-15-9560",
			application: "org.gnome.Shell.desktop",
			text:        "== Stack trace for context 0x563cea7c7340 ==",
		},
		{
			raw:         []byte("<6>1 2018-08-09T07:19:28.698693Z myhost myapp - - - it is all fucked"),
			time:        time.Date(2018, 8, 9, 07, 19, 28, 698693000, time.UTC),
			hostname:    "myhost",
			application: "myapp",
			text:        "it is all fucked",
		},
		{
			raw:         []byte("<14>2018-06-19T11:08:00-07:00 bar elasticsearch: [2018-06-19 11:08:00,000][DEBUG][gateway] [Blizzard II] recovered [0] indices into cluster_state foo=bar"),
			time:        time.Date(2018, 6, 19, 18, 8, 00, 0, time.UTC),
			hostname:    "bar",
			application: "elasticsearch",
			text:        "[2018-06-19 11:08:00,000][DEBUG][gateway] [Blizzard II] recovered [0] indices into cluster_state foo=bar",
			metadata: map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			raw:         []byte(`<6> Mar  7 05:45:39 eth systemd[1]: Starting Message of the Day...`),
			time:        time.Date(time.Now().Year(), 3, 7, 5, 45, 39, 0, time.UTC),
			hostname:    "eth",
			application: "systemd",
			text:        "Starting Message of the Day...",
			adjust:      true,
		},
	}

	for _, c := range cases {
		msg := ParseLineWithFallback(c.raw, "0.0.0.0")
		str := string(c.raw)
		require.NotNil(msg)

		if c.time.Year() != 0 {
			ts := time.Unix(0, msg.Timestamp)
			if !c.adjust {
				ts = ts.In(c.time.Location())
			}
			require.Equal(c.time.Year(), ts.Year(), str)
			require.Equal(c.time.Month(), ts.Month(), str)
			require.Equal(c.time.Day(), ts.Day(), str)
			require.Equal(c.time.Hour(), ts.Hour(), str)
			require.Equal(c.time.Minute(), ts.Minute(), str)
			if !c.imprecise {
				require.Equal(c.time.Second(), ts.Second(), str)
				require.Equal(c.time.Nanosecond(), ts.Nanosecond(), str)
			}
		}

		if c.hostname != "" {
			require.Equal(c.hostname, msg.Hostname, str)
		}
		require.Equal(c.application, msg.Application, str)
		require.Equal(c.text, msg.Text, str)

		for k, v := range c.metadata {
			require.Equal(v, msg.Metadata[k], fmt.Sprintf("metadata mismatch on key '%s': %+v\n", k, msg.Metadata))
		}

		if c.metadataLTLen > 0 {
			require.True(len(msg.Metadata) < c.metadataLTLen, fmt.Sprintf("Metadata expectation failed: length: %d, raw: %s", len(msg.Metadata), str))
		}
	}
}

func (s *ParseTestSuite) TestRFC3164Dates() {
	assert := assert.New(s.T())

	type syslogFixture struct {
		raw   []byte
		isUTC bool
	}
	fixtures := []syslogFixture{
		{raw: []byte("<34>Oct 1 22:14:15 mymachine very.large.syslog.message.tag[2400]: 'su root' failed for lonvick on /dev/pts/8")},
		{raw: []byte("<34>Oct  1 22:14:15 mymachine very.large.syslog.message.tag[2400]: 'su root' failed for lonvick on /dev/pts/8")},
		{raw: []byte("<34>Oct 01 22:14:15 mymachine very.large.syslog.message.tag[2400]: 'su root' failed for lonvick on /dev/pts/8")},
		{raw: []byte(fmt.Sprintf("<34>%d-10-01T22:14:15Z mymachine very.large.syslog.message.tag[2400]: 'su root' failed for lonvick on /dev/pts/8", time.Now().Year())), isUTC: true},
		{raw: []byte(fmt.Sprintf("<34>%d-10-01T22:14:15+00:00 mymachine very.large.syslog.message.tag[2400]: 'su root' failed for lonvick on /dev/pts/8", time.Now().Year())), isUTC: true},
	}

	for _, f := range fixtures {
		str := string(f.raw)
		msg, _ := parseLine(f.raw)
		assert.NotNil(msg)
		assert.Equal(int64(2), msg.Severity, str)
		loc := time.Local
		if f.isUTC {
			loc = time.UTC
		}
		expectedTS := time.Date(time.Now().Year(), time.October, 1, 22, 14, 15, 0, loc)
		assert.Equal(expectedTS, time.Unix(0, msg.Timestamp).In(expectedTS.Location()), str)
		assert.Equal("mymachine", msg.Hostname, str)
		assert.Equal("very.large.syslog.message.tag", msg.Application, str)
		assert.Equal("'su root' failed for lonvick on /dev/pts/8", msg.Text, str)
	}
}

func (s *ParseTestSuite) TestRFC3164SequenceID() {
	assert := assert.New(s.T())

	buff := []byte("<34>214: Oct 11 22:14:15 mymachine very.large.syslog.message.tag: 'su root' failed for lonvick on /dev/pts/8")

	msg, _ := parseLine(buff)
	assert.NotNil(msg)
	assert.Equal("214", msg.Metadata["SequenceID"])
	assert.Equal(time.Date(time.Now().Year(), time.October, 11, 22, 14, 15, 0, time.Local), time.Unix(0, msg.Timestamp))
	assert.Equal("'su root' failed for lonvick on /dev/pts/8", msg.Text)
}

func (s *ParseTestSuite) TestRFC3164NoTimeOrHost() {
	assert := assert.New(s.T())

	buff := []byte("<34>214: myprogram[332] 'su root' failed for lonvick on /dev/pts/8")

	msg, _ := parseLine(buff)
	assert.NotNil(msg)
	assert.Equal("214", msg.Metadata["SequenceID"])
	assert.Equal(time.Now().UTC().Day(), time.Unix(0, msg.Timestamp).UTC().Day())
	assert.Equal("'su root' failed for lonvick on /dev/pts/8", msg.Text)
}

func (s *ParseTestSuite) TestSynthetic() {
	msg := ParseLineWithFallback([]byte("foobar2000"), "127.0.0.1")
	s.Equal("foobar2000", msg.Text)

	msg = ParseLineWithFallback([]byte("<14>sourcehost tag text"), "127.0.0.1")
	s.Require().NotNil(msg)
	s.Equal("sourcehost", msg.Hostname)
	s.Equal("tag", msg.Application)
	s.Equal("text", msg.Text)
}

func (s *ParseTestSuite) TestSyntheticDirect() {
	msg, _ := syntheticLog("myhost", []byte("This is a message"))
	s.Require().NotNil(msg)
	s.Equal("This is a message", msg.Text)
	s.Equal("myhost", msg.Hostname)
	s.Equal("unknown", msg.Application)
}

func (s *ParseTestSuite) TestFuzzCrashers() {
	payloads := [][]byte{
		[]byte("<>:"),
		[]byte("<00"),
	}

	for _, data := range payloads {
		_, err := parseLine(data)
		s.Error(err)
	}
}

func (s *ParseTestSuite) TestUnescape() {
	assert := assert.New(s.T())

	buff := []byte("<34>214: myprogram[332]: #033[32mdebug#033[0m #033[37;2mdatastores#033[0m@#033[94mdatastores.statsd#033[0m accumulator.go:149 Encountered err #033")

	msg, _ := parseLine(buff)
	assert.NotNil(msg)
	assert.Equal("\x1b[32mdebug\x1b[0m \x1b[37;2mdatastores\x1b[0m@\x1b[94mdatastores.statsd\x1b[0m accumulator.go:149 Encountered err #033", msg.Text)
}

func (s *ParseTestSuite) TestExtractSeverity() {
	assert := s.Assert()

	cases := map[string]int32{
		"critical": Critical,
		"error":    Error,
		"warn":     Warning,
		"info":     Info,
		"debug":    Debug,
		"trace":    Trace,
	}

	for t, s := range cases {
		mixed := t[:2] + (strings.ToUpper(t)[2:])
		tests := []string{t, strings.ToUpper(t), mixed}

		for _, test := range tests {
			start := fmt.Sprintf("%s: all good", test)
			middle := fmt.Sprintf("all %s good", test)
			end := fmt.Sprintf("all good %s", test)

			assert.EqualValues(s, extractSeverity(start), start)
			assert.EqualValues(s, extractSeverity(middle), middle)
			assert.EqualValues(s, extractSeverity(end), end)
			assert.EqualValues(Unknown, extractSeverity(test[:2]))
			assert.EqualValues(Unknown, extractSeverity("foo "+test[:2]+" wat"))
			assert.EqualValues(Unknown, extractSeverity("this is so "+test[:2]))
		}
	}
}

func Benchmark5424(b *testing.B) {
	raw := []byte("<134>1 2009-10-16T11:51:56+02:00 ip-34-23-211-23 symbolicator 2008 SOMEMSG - hello")
	for i := 0; i < b.N; i++ {
		msg, _ := parseLine(raw)
		if msg == nil {
			panic(errors.New("Unable to parse message"))
		}
	}
}

func BenchmarkParserDifferentMetadataTypes(b *testing.B) {
	raw := []byte(`<14> src time="2018-06-02T17:16:14.392415523+01:00" bool=false level=info float=5.6 number=3 msg="[graphdriver] using prior storage driver: aufs"`)
	for i := 0; i < b.N; i++ {
		msg, _ := parseLine(raw)
		if msg == nil {
			panic(errors.New("Unable to parse message"))
		}
	}
}

func BenchmarkParser(b *testing.B) {
	p := New(func(msg *Log) {})

	const msg = "<1> 2009-10-16T11:51:56+02:00 ip-34-23-211-23 symbolicator ERROR 2008 SOMEMSG - hello"
	rawMsg := []byte(msg)
	raw := []byte("<999>h9:f6:m" + strconv.Itoa(len(msg)) + " localhost syslog " + msg)

	for i := 0; i < b.N; i++ {
		// agent path
		p.WriteLine(raw, "127.0.0.1")
		// syslog path
		p.WriteLine(rawMsg, "127.0.0.1")
	}
}

func BenchmarkDateParse(b *testing.B) {
	raw := []byte("<13>Jan  1 14:40:51 host app[24]: this is the message")
	for i := 0; i < b.N; i++ {
		msg, _ := parseLine(raw)
		if msg == nil {
			panic(errors.New("Unable to parse message"))
		}
	}
}

func BenchmarkNoDateParse(b *testing.B) {
	raw := []byte("<13>host app[24]: this is the message")
	for i := 0; i < b.N; i++ {
		msg, _ := parseLine(raw)
		if msg == nil {
			panic(errors.New("Unable to parse message"))
		}
	}
}
