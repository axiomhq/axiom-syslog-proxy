package parser

// Parser ...
type Parser interface {
	WriteLine(line []byte, remoteIP string)
	Flush() error
	Stop() error
}

// ProcessLogFunc ...
type ProcessLogFunc func(msg *Log)

type parser struct {
	emitLog ProcessLogFunc
}

// New ...
func New(cb ProcessLogFunc) Parser {
	return &parser{
		emitLog: cb,
	}
}

func (p *parser) WriteLine(line []byte, remoteIP string) {
	// This is obviously the simple case for now, but with the `parser` type
	// we'll be able to:
	// a) Be able to take into account the specific log parsing settings of the instance and,
	// b) Intiialize & involve integrations for parsing specific log types
	if msg := ParseLineWithFallback(line, remoteIP); msg != nil {
		if msg.Text == "" {
			return
		}

		p.emitLog(msg)
	}
}

func (p *parser) Flush() error {
	return nil
}

func (p *parser) Stop() error {
	return nil
}
