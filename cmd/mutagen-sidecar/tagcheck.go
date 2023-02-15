//go:build !mutagensidecar

package main

func init() {
	panic("executable built with without correct tag")
}
