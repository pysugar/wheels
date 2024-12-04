package client

import (
	"context"
	"log"
)

type (
	VerboseLogger interface {
		Verbose() bool
		Printf(format string, v ...any)
		Println(v ...any)
	}

	verboseLogger struct {
		verbose bool
	}
)

func newVerboseLogger(ctx context.Context) VerboseLogger {
	return &verboseLogger{
		verbose: VerboseFromContext(ctx),
	}
}

func (vl *verboseLogger) Verbose() bool {
	return vl.verbose
}

func (vl *verboseLogger) Printf(format string, v ...any) {
	if vl.verbose {
		log.Printf(format, v...)
	}
}

func (vl *verboseLogger) Println(v ...any) {
	if vl.verbose {
		log.Println(v...)
	}
}
