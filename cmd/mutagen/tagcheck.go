//go:build !mutagencli

package main

func init() {
	panic("executable built with without correct tag")
}
