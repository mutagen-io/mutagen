//go:build mutagensspl && !mutagencli

package hashing

// xxh128SupportStatus returns XXH128 hashing support status.
func xxh128SupportStatus() AlgorithmSupportStatus {
	return AlgorithmSupportStatusSupported
}
