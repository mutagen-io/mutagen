package grpcutil

import (
	"io/ioutil"
	"log"

	"google.golang.org/grpc/grpclog"
)

// Squelch silences gRPC log output. It must be only be called in init.
func Squelch() {
	grpclog.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags))
}
