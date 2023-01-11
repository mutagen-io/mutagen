package synchronization

import (
	"testing"
)

// TestDigestUnmarshal tests that unmarshaling from a string specification
// succeeeds for Digest.
func TestDigestUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text           string
		expectedDigest Digest
		expectFailure  bool
	}{
		{"", Digest_DigestDefault, true},
		{"asdf", Digest_DigestDefault, true},
		{"sha1", Digest_DigestSHA1, false},
		{"sha256", Digest_DigestSHA256, false},
		{"xxh128", Digest_DigestXXH128, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var digest Digest
		if err := digest.UnmarshalText([]byte(testCase.text)); err != nil {
			if !testCase.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.text, err)
			}
		} else if testCase.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.text)
		} else if digest != testCase.expectedDigest {
			t.Errorf(
				"unmarshaled digest (%s) does not match expected (%s)",
				digest,
				testCase.expectedDigest,
			)
		}
	}
}

// TestDigestSupported tests that Digest support detection works as expected.
func TestDigestSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		digest          Digest
		expectSupported bool
	}{
		{Digest_DigestDefault, false},
		{Digest_DigestSHA1, true},
		{Digest_DigestSHA256, true},
		{Digest_DigestXXH128, digestXXH128Supported},
		{(Digest_DigestXXH128 + 1), false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.digest.Supported(); supported != testCase.expectSupported {
			t.Errorf(
				"digest support status (%t) does not match expected (%t)",
				supported,
				testCase.expectSupported,
			)
		}
	}
}

// TestDigestDescription tests that Digest description generation works as
// expected.
func TestDigestDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		digest              Digest
		expectedDescription string
	}{
		{Digest_DigestDefault, "Default"},
		{Digest_DigestSHA1, "SHA-1"},
		{Digest_DigestSHA256, "SHA-256"},
		{Digest_DigestXXH128, "XXH128"},
		{(Digest_DigestXXH128 + 1), "Unknown"},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if description := testCase.digest.Description(); description != testCase.expectedDescription {
			t.Errorf(
				"digest description (%s) does not match expected (%s)",
				description,
				testCase.expectedDescription,
			)
		}
	}
}
