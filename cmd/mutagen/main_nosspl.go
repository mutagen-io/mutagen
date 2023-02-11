//go:build !sspl

package main

// registerLicenseCommand registers the license command tree with the root
// command. This function is a no-op in non-SSPL builds.
func registerLicenseCommand() {}
