package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Logger struct {
	sb       *strings.Builder
	filePath string
}

func (logger Logger) Writeln(lineNumber int, str string, args ...any) {
	logger.sb.WriteString(logger.filePath)
	logger.sb.WriteString(":")
	if lineNumber != 0 {
		logger.sb.WriteString(strconv.Itoa(lineNumber))
	}
	logger.sb.WriteString(" ")
	fmt.Fprintf(logger.sb, str, args...)
	logger.sb.WriteString("\n")
}

func (logger Logger) String() string {
	return logger.sb.String()
}
