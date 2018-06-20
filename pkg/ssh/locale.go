package ssh

// addLocaleVariables adds environment variables to try to coax SSH servers into
// using an encoding that is (a) widely supported and (b) easy to support on our
// end. Essentially it tries to get the server to spit out nothing that isn't
// UTF-8. It's worth noting that this will also affect OpenSSH's behavior, but
// since OpenSSH doesn't support internationalization at the moment, it's sort
// of irrelevant.
func addLocaleVariables(environment []string) []string {
	// Set the LANG and all LC_ variables to a C (ASCII) locale (note that
	// LC_ALL takes precedence over LANG as well). This is the safest possible
	// option. If for some reason we start needing Unicode support in the very
	// few commands that we run (which I can't imagine we will), we can try to
	// set this to en_US.UTF-8.
	return append(environment, "LC_ALL=C")
}
