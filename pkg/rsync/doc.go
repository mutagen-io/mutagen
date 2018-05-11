// Package rsync provides an implementation of the rsync algorithm as described
// in Andrew Tridgell's thesis (https://www.samba.org/~tridge/phd_thesis.pdf)
// and the rsync technical report (https://rsync.samba.org/tech_report). Rsync
// algorithmic functionality is provided by the Engine type, and a transport
// protocol for pipelined rsync operations is provided by the Client and Server
// types.
package rsync
