package input

import "github.com/axiomhq/logmanager"

var logger = logmanager.GetLogger("logs/input")

// WriteLineFunc ...
type WriteLineFunc func(line []byte, remoteIP string)
