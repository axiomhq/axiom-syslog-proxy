package parser

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"
)

const (
	// RFC 5424 pre-defined SD-IDs
	sdIDTimeQuality = "timeQuality"
)

var (
	stdFormats = []string{
		"Jan 02 2006 15:04:05",
		"Jan 02 15:04:05.000",
		"Jan 02 15:04:05.00",
		"Jan 02 15:04:05.0",
		"Jan 02 15:04:05",
		"Jan  2 15:04:05",
		"Jan 2 15:04:05",
	}
	isoFormats = []string{
		// golang will parse even fractional seconds with these formats
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	metadataKeyRegex = regexp.MustCompile(`^\w(?:\w|\s|[.-])*$`)
	termCodesRegex   = regexp.MustCompile("#033[[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-PRZcf-nqry=><]")
	// these are #011, #012, #015 (TAB, LF, CR)
	escapedCtrlCharsRegex = regexp.MustCompile(`#01[125]`)

	errParse         = errors.New("parsing error")
	errCorruptedData = errors.New("corrupted data")
)

type dateFormatType int

const (
	dateFormatAny dateFormatType = iota
	dateFormatISO
)

func parseSyslog(data []byte) (*Log, error) {
	msg := &Log{
		Severity: Unknown,
		Metadata: map[string]interface{}{},
	}

	length := len(data)
	for length > 0 && (data[length-1] == '\n' || data[length-1] == '\x00') {
		length--
	}

	if length < 3 {
		return nil, errParse
	}

	var parseErr error
	if parseErr = parseRFC5424(msg, data, length); parseErr == errParse {
		parseErr = parseRFC3164(msg, data, length)
	}
	if parseErr != nil {
		return nil, parseErr
	}

	return msg, nil
}

func parseMetadata(msg *Log, data []byte) {
	max := len(data)
	if max < 3 {
		return
	}

	i := 0
	for i < max {
		if data[i] == '=' && i-1 > 0 && i+1 < max {
			if key := findKey(data[:i]); key != "" && metadataKeyRegex.MatchString(key) {
				if val := findVal(data[i+1:]); val != nil {
					success := false

					if len(val) > 0 {
						if f := val[0]; f >= '0' && f <= '9' {
							if i, err := ParseInt(val); err == nil {
								msg.Metadata[cleanString(key, true)] = i
								success = true
							} else if f, err := ParseFloat(val); err == nil {
								msg.Metadata[cleanString(key, true)] = f
								success = true
							}
						}
					}
					// Not sure what it is, cast it to string, done.
					if !success {
						msg.Metadata[cleanString(key, true)] = cleanString(string(val), true)
					}
					i += 1 + len(val)
					continue
				}
			}
		}

		i++
	}
}

func findKey(part []byte) string {
	end := len(part) - 1
	i := end
	quoted := false

	switch part[i] {
	case ' ':
		return ""
	case '"':
		i--
		quoted = true
	}

	// We look backwards for the key
	for i >= 0 {
		switch part[i] {
		case ' ':
			if !quoted {
				return string(part[i+1:])
			}

		case '"':
			if quoted {
				return string(part[i+1 : end])
			}
		}

		i--

		if i < 0 && !quoted {
			return string(part)
		}
	}

	return ""
}

func findVal(part []byte) []byte {
	max := len(part)
	start := 0
	i := 0
	quoted := false

	switch part[i] {
	case ' ':
		return nil
	case '"':
		i++
		start++
		quoted = true
	}

	// We look forwards for the key
	for i < max {
		switch part[i] {
		case ' ':
			if !quoted {
				return part[start:i]
			}

		case '"':
			if quoted && i > 0 && part[i-1] != '\\' {
				return part[start:i]
			}
		}

		i++

		if i >= max && !quoted {
			return part[start:]
		}
	}

	return nil
}

func parseRFC3164(msg *Log, data []byte, length int) error {
	i := 0
	l := length

	if !parsePriority(msg, data, &i, &l) {
		return errParse
	}

	parseSequenceID(msg, data, &i, &l)
	skipChar(data, &i, &l, ' ', -1)

	if parseDate(msg, dateFormatAny, data, &i, &l) {
		skipChar(data, &i, &l, ' ', -1)
	} else {
		msg.Timestamp = time.Now().UnixNano()
	}

	// Expected: `hostname program[pid]:` though both are optional
	parseHostname(msg, data, &i, &l)
	skipChar(data, &i, &l, ' ', -1)
	parse3164Application(msg, data, &i, &l)

	// Sometimes we'll catch in hostname instead of app
	if msg.Hostname != "" && msg.Application == "" {
		msg.Application = msg.Hostname
		msg.Hostname = ""
	}

	valid, textData := processText(data[i:])
	if !valid {
		return errCorruptedData
	}
	msg.Text = cleanString(textData, false)

	parseMetadata(msg, data[i:])

	return nil
}

func parseRFC5424(msg *Log, data []byte, length int) error {
	// SYSLOG-MSG: HEADER SP STRUCTURED-DATA [SP MSG]
	// HEADER: PRI VERSION SP TIMESTAMP SP HOSTNAME SP APP-NAME SP PROCID SP MSGID
	i := 0
	l := length

	if !parsePriority(msg, data, &i, &l) || !parseVersion(data, &i, &l) {
		return errParse
	}

	if !skipSpace(data, &i, &l) {
		return errParse
	}

	if !parseDate(msg, dateFormatISO, data, &i, &l) {
		return errParse
	}

	parseHostname(msg, data, &i, &l)
	if !skipSpace(data, &i, &l) {
		return errParse
	}

	msg.Application = string(parseColumn(data, &i, &l))
	if !skipSpace(data, &i, &l) {
		return errParse
	}

	// procid
	parseColumn(data, &i, &l)
	if !skipSpace(data, &i, &l) {
		return errParse
	}

	// msgid
	parseColumn(data, &i, &l)
	if !skipSpace(data, &i, &l) {
		return errParse
	}

	// If no structured data, then move the point along to miss the "- "
	if l > 0 && data[i] == '-' {
		i++
		l--
	} else {
		if sd, sdErr := parseStructuredData(data, &i, &l); sdErr == nil {
			for sdID, sdEl := range sd {
				if sdID == sdIDTimeQuality {
					// do we really need to index this?
					continue
				}

				var prefix string
				if strings.HasPrefix(sdID, "axiom") {
					prefix = ""
				} else if idx := strings.IndexRune(sdID, '@'); idx > 0 {
					prefix = sdID[0:idx] + "."
				} else {
					prefix = sdID + "."
				}

				for param, val := range sdEl {
					msg.Metadata[prefix+param] = val
				}
			}
		} else {
			return sdErr
		}
	}
	// optional space after SD
	skipSpace(data, &i, &l)

	msg.Application = parseApplication(msg.Application)
	valid, textData := processText(data[i:])
	if !valid {
		return errCorruptedData
	}
	msg.Text = cleanString(textData, false)

	parseMetadata(msg, data[i:])

	return nil
}

func cleanString(s string, unquote bool) string {
	if unquote {
		if unquoted, _ := strconv.Unquote(s); unquoted != "" {
			s = unquoted
		}
	}
	return strings.Replace(s, "\\", "", -1)
}

func parseApplication(app string) string {
	if n := strings.Index(app, "["); n >= 0 {
		return app[:n]
	}
	return app
}

func parseColumn(data []byte, index *int, length *int) []byte {
	i := *index
	l := *length

	var space int
	var j int
	for j = 0; j < l; j++ {
		if data[i+j] == ' ' {
			space = j
			break
		}
	}

	if space > 0 {
		i += space
		l -= space
	} else {
		i += l
		l = 0
	}

	var result []byte

	if l > 0 {
		if *length-l > 0 {
			result = data[*index:i]
		}
	}

	if len(result) == 1 && result[0] == '-' {
		result = nil
	}

	*index = i
	*length = l

	return result
}

func skipSpace(data []byte, index *int, length *int) bool {
	i := *index
	l := *length

	if l > 0 && data[i] == ' ' {
		*index = i + 1
		*length = l - 1
		return true
	}

	return false
}

func parseVersion(data []byte, index *int, length *int) bool {
	i := *index
	l := *length
	var version int32

	for l > 0 && data[i] != ' ' {
		if unicode.IsDigit(rune(data[i])) {
			version = version*10 + (rune(data[i]) - '0')
		} else {
			return false
		}
		i++
		l--
	}
	if version <= 0 || version > 999 {
		return false
	}

	*index = i
	*length = l
	return true
}

func parse3164Application(msg *Log, data []byte, index *int, length *int) bool {
	i := *index
	l := *length

	for l > 0 && data[i] != ' ' && data[i] != '[' && data[i] != ':' {
		i++
		l--
	}

	// tag can't exceed 32 chars
	if i-*index > 32 {
		return false
	}

	app := string(data[*index:i])

	// Check for PID
	if l > 0 && data[i] == '[' {
		for l > 0 && data[i] != ' ' && data[i] != ']' && data[i] != ':' {
			i++
			l--
		}
		if l > 0 && data[i] == ']' {
			i++
			l--
		}
	}

	if l > 0 && data[i] == ':' {
		i++
		l--
	}

	spaceIdx := i

	if l > 0 && data[i] == ' ' {
		i++
		l--
	}

	if i == spaceIdx {
		// no space after the tag, so most likely not a tag
		return false
	}

	msg.Application = app

	*index = i
	*length = l

	return true
}

func parseHostname(msg *Log, data []byte, index *int, length *int) {
	i := *index
	l := *length

	for l > 0 && data[i] != ' ' {
		i++
		l--
	}

	if data[i-1] == ':' || data[i-1] == ']' {
		// If we encounter this, we're actually parsing the application
		return
	}

	msg.Hostname = string(data[*index:i])

	*index = i
	*length = l
}

func parseDate(msg *Log, dateFormat dateFormatType, data []byte, index *int, length *int) bool {
	i := *index
	l := *length

	loc := time.Local

	formats := stdFormats
	timeStrLen := -1
	suffixLen := 0

	for j := i; j < i+l && data[j] != ' '; j++ {
		if data[j] == 'Z' || data[j] == '+' || data[j] == '-' {
			formats = isoFormats
			if spaceIdx := bytes.IndexByte(data[j:], ' '); spaceIdx > 0 {
				timeStrEnd := j + spaceIdx
				timeStrLen = timeStrEnd - i

				// not sure why golang doesn't deal with this automatically
				if isZulu := data[timeStrEnd-1] == 'Z'; isZulu {
					loc = time.UTC
					timeStrLen--
					suffixLen++

					// and jump straight to the format that will work
					formats = isoFormats[len(isoFormats)-1:]
				}
			} else {
				// a valid packet would always have a space, so we know we're parsing something invalid
				// nonetheless the time parsing might still work...
				timeStrLen = l
			}
			break
		}
	}

	// if timeStrLen wasn't set, we didn't detect valid iso format
	if dateFormat == dateFormatISO && timeStrLen <= 0 {
		return false
	}

	// the formats get progressively shorter, so we can avoid multiple allocs
	var timeStr string
	if timeStrLen > 0 {
		timeStr = string(data[i : i+timeStrLen])
	} else {
		maxIdx := i + len(formats[0])
		if maxIdx > len(data) {
			maxIdx = len(data)
		}
		timeStr = string(data[i:maxIdx])
	}

	for _, format := range formats {
		fmtLen := len(format)

		if timeStrLen > 0 {
			fmtLen = timeStrLen
		} else if fmtLen > l {
			continue
		}
		s := timeStr[:fmtLen]
		ts, err := time.ParseInLocation(format, s, loc)
		if err == nil {
			if ts.Year() == 0 {
				ts = ts.AddDate(time.Now().Year(), 0, 0)
			}
			msg.Timestamp = ts.UnixNano()

			i += fmtLen + suffixLen
			l -= fmtLen + suffixLen

			if l > 0 {
				i++
				l--
			}

			*index = i
			*length = l
			return true
		}
	}
	return false
}

func parseStructuredData(data []byte, index *int, length *int) (map[string]map[string]string, error) {
	offset := *index

	if data[offset] != '[' {
		return nil, errParse
	}

	result := map[string]map[string]string{}

	sc := scanner.Scanner{}
	sc.Init(bytes.NewBuffer(data[offset+1:]))
	sc.Mode = scanner.ScanStrings | scanner.ScanIdents
	sc.IsIdentRune = func(ch rune, _ int) bool {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '@' || ch == '.'
	}
	sc.Error = func(_ *scanner.Scanner, _ string) {}

	var (
		sdID, sdParam string
		expectingVal  bool
		inSdElem      = true
		sdEndIdx      = -1
	)

scannerLoop:
	for tok := sc.Scan(); tok != scanner.EOF; tok = sc.Scan() {
		switch tok {
		case scanner.String:
			text := sc.TokenText()
			if expectingVal {
				if len(text) < 2 {
					return nil, errParse
				}
				unqouted := text[1 : len(text)-1]
				if idx := strings.IndexRune(unqouted, '\x00'); idx >= 0 {
					// trim the values if they contain null bytes
					unqouted = unqouted[:idx]
				}
				result[sdID][sdParam] = cleanString(unqouted, false)

				sdParam = ""
			} else {
				break scannerLoop
			}
		case scanner.Ident:
			ident := sc.TokenText()
			if sdID == "" {
				sdID = ident
				result[sdID] = map[string]string{}
			} else if sdParam == "" {
				sdParam = ident
			} else {
				break scannerLoop
			}
		case '[':
			if !inSdElem {
				inSdElem = true
			} else {
				break scannerLoop
			}
		case ']':
			if inSdElem {
				inSdElem = false
				sdID = ""
			} else {
				break scannerLoop
			}

			nextIdx := offset + 1 + sc.Position.Offset + 1
			if nextIdx >= len(data) {
				// the loop will end with EOF
				sdEndIdx = nextIdx
				continue
			}
			nextCh := data[nextIdx]
			if nextCh != '[' {
				sdEndIdx = nextIdx
				break scannerLoop
			}
		case '=':
			if sdParam != "" {
				expectingVal = true
			} else {
				break scannerLoop
			}
		}
	}

	if sdEndIdx > 0 {
		*index = sdEndIdx
		*length = len(data) - sdEndIdx
		return result, nil
	}

	return nil, errParse
}

func skipChar(data []byte, index *int, length *int, char byte, maxSkip int) {
	i := *index
	l := *length

	for maxSkip != 0 && l > 0 && data[i] == char {
		i++
		l--
		if maxSkip >= 0 {
			maxSkip--
		}
	}
	*index = i
	*length = l
}

func processText(data []byte) (bool, string) {
	trimmed := trimBOM(data)
	// some syslog forwarders escape control chars, check for that (especially the ESC key code)
	if bytes.Contains(trimmed, []byte("#033")) {
		// and unescape, so our term char filter works properly
		trimmed = termCodesRegex.ReplaceAllFunc(trimmed, func(code []byte) []byte {
			out := make([]byte, len(code)-3)
			out[0] = 0x1b
			copy(out[1:], code[4:])
			return out
		})
	}
	if idx := bytes.Index(trimmed, []byte("#01")); idx >= 0 && idx+3 < len(trimmed) {
		trimmed = escapedCtrlCharsRegex.ReplaceAllFunc(trimmed, func(code []byte) []byte {
			oct, err := ParseInt(code[1:])
			if err != nil {
				return code
			}
			ch := (oct / 10 * 8) + oct%10
			return []byte{byte(ch)}
		})
	}

	// trim anything past the first null byte
	if idx := bytes.IndexByte(trimmed, 0); idx >= 0 {
		// if the null byte is first, we'll just ignore the whole packet
		return idx > 0, string(trimmed[:idx])
	}

	return true, string(trimmed)
}

// Check for unicode BOM because gord reads RFCs
func trimBOM(trimmed []byte) []byte {
	if len(trimmed) > 3 && trimmed[0] == 0xef && trimmed[1] == 0xbb && trimmed[2] == 0xbf {
		return bytes.TrimSpace(trimmed[3:])
	}
	return trimmed
}

func parseSequenceID(msg *Log, data []byte, index *int, length *int) bool {
	i := *index
	l := *length

	for l > 0 && data[i] != ':' {
		if !unicode.IsDigit(rune(data[i])) {
			return false
		}
		i++
		l--
	}
	i++
	l--

	if i >= len(data) || data[i] != ' ' {
		return false
	}

	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["SequenceID"] = string(data[*index : i-1])

	*index = i
	*length = l
	return true
}

func parsePriority(msg *Log, data []byte, index *int, length *int) bool {
	i := *index
	l := *length

	if l > 0 && data[i] == '<' {
		i++
		l--
		var pri int32
		validPri := false
		for l > 0 && data[i] != '>' {
			if unicode.IsDigit(rune(data[i])) {
				validPri = true
				pri = pri*10 + (rune(data[i]) - '0')
			} else {
				return false
			}
			i++
			l--
		}

		if !validPri {
			return false
		} else if i >= len(data) || data[i] != '>' {
			return false
		}

		msg.Severity = int64(pri % 8)

		if l > 0 {
			i++
			l--
		}
	}
	*index = i
	*length = l
	return true
}
