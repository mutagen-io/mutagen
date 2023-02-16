//go:build !mutagenagent

package main

func init() {
	panic("executable built with without correct tag")
}
