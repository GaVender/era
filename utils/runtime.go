package utils

import (
	"bytes"
	"runtime"
	"strconv"
)

func Stack(skip int) (s bytes.Buffer) {
	stack := make([]uintptr, 10)
	high := runtime.Callers(skip, stack)
	f := runtime.CallersFrames(stack[:high])

	if f != nil {
		for {
			nf, ok := f.Next()
			if !ok {
				break
			}

			s.WriteString(nf.File)
			s.WriteRune(':')
			s.WriteString(strconv.Itoa(nf.Line))
			s.WriteString("\n")
		}
	}

	return s
}
