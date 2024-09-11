package app

import (
	"fmt"
	"io"
	"os"
)

type Logger struct {
	out io.Writer
	err io.Writer
}

func NewLogger(outWriter io.Writer, errWriter io.Writer) *Logger {
	return &Logger{
		out: outWriter,
		err: errWriter,
	}
}

func (l *Logger) Log(def io.Writer, s string, a ...any) {
	fmt.Fprintln(def, fmt.Sprintf(s, a...))
}

func (l *Logger) Info(s string, a ...any) {
	l.Log(l.out, s, a...)
}

func (l *Logger) Error(s string, a ...any) {
	l.Log(l.err, s, a...)
}

func (l *Logger) Fatal(s string, a ...any) {
	l.Log(l.err, s, a...)
	os.Exit(1)
}
