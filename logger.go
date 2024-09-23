package gosiklu

import (
	"fmt"
	"github.com/google/goterm/term"
)

type clientLogger struct{}

func (l *clientLogger) Errorf(format string, v ...interface{}) {
	fmt.Println(term.Redf(format, v...))
}

func (l *clientLogger) Warnf(format string, v ...interface{}) {
	fmt.Println(term.Yellowf(format, v...))
}

func (l *clientLogger) Debugf(format string, v ...interface{}) {
	fmt.Println(term.Cyanf(format, v...))
}
