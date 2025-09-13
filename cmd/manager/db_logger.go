package main

import (
	"fmt"
	"log/slog"
	"strings"
)

type slogBadgerAdapter struct {
}

func (l slogBadgerAdapter) Errorf(f string, details ...any) {
	for line := range strings.SplitSeq(fmt.Sprintf(f, details...), "\n") {
		if line != "" {
			slog.Error("badger", "msg", line)
		}
	}
}

func (l slogBadgerAdapter) Warningf(f string, details ...any) {
	for line := range strings.SplitSeq(fmt.Sprintf(f, details...), "\n") {
		if line != "" {
			slog.Warn("badger", "msg", line)
		}
	}
}

func (l slogBadgerAdapter) Infof(f string, details ...any) {
	for line := range strings.SplitSeq(fmt.Sprintf(f, details...), "\n") {
		if line != "" {
			slog.Info("badger", "msg", line)
		}
	}
}

func (l slogBadgerAdapter) Debugf(f string, details ...any) {
	for line := range strings.SplitSeq(fmt.Sprintf(f, details...), "\n") {
		if line != "" {
			slog.Debug("badger", "msg", line)
		}
	}
}
