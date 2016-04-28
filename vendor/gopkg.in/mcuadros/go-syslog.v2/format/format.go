package format

import (
	"bufio"

	"gopkg.in/mcuadros/go-syslog.v2/internal/syslogparser"
)

type Format interface {
	GetParser([]byte) syslogparser.LogParser
	GetSplitFunc() bufio.SplitFunc
}
