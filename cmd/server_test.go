package cmd

import (
	"testing"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
)

func Test_checkError(t *testing.T) {
	type args struct {
		err    error
		logger *loggers.Logger
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkError(tt.args.err, tt.args.logger)
		})
	}
}
