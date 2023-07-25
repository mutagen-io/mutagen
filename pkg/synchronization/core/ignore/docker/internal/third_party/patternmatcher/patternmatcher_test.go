// Tests for pattern matching infrastructure used to support .dockerignore
// files. Based on (but modified from)
// https://github.com/moby/patternmatcher/blob/c5e4b22c8cb290f9439a339c08bba6cb13aa296d/patternmatcher_test.go
//
// The original code license:
//
//                               Apache License
//                         Version 2.0, January 2004
//                      https://www.apache.org/licenses/
//
// TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION
//
// 1. Definitions.
//
//    "License" shall mean the terms and conditions for use, reproduction,
//    and distribution as defined by Sections 1 through 9 of this document.
//
//    "Licensor" shall mean the copyright owner or entity authorized by
//    the copyright owner that is granting the License.
//
//    "Legal Entity" shall mean the union of the acting entity and all
//    other entities that control, are controlled by, or are under common
//    control with that entity. For the purposes of this definition,
//    "control" means (i) the power, direct or indirect, to cause the
//    direction or management of such entity, whether by contract or
//    otherwise, or (ii) ownership of fifty percent (50%) or more of the
//    outstanding shares, or (iii) beneficial ownership of such entity.
//
//    "You" (or "Your") shall mean an individual or Legal Entity
//    exercising permissions granted by this License.
//
//    "Source" form shall mean the preferred form for making modifications,
//    including but not limited to software source code, documentation
//    source, and configuration files.
//
//    "Object" form shall mean any form resulting from mechanical
//    transformation or translation of a Source form, including but
//    not limited to compiled object code, generated documentation,
//    and conversions to other media types.
//
//    "Work" shall mean the work of authorship, whether in Source or
//    Object form, made available under the License, as indicated by a
//    copyright notice that is included in or attached to the work
//    (an example is provided in the Appendix below).
//
//    "Derivative Works" shall mean any work, whether in Source or Object
//    form, that is based on (or derived from) the Work and for which the
//    editorial revisions, annotations, elaborations, or other modifications
//    represent, as a whole, an original work of authorship. For the purposes
//    of this License, Derivative Works shall not include works that remain
//    separable from, or merely link (or bind by name) to the interfaces of,
//    the Work and Derivative Works thereof.
//
//    "Contribution" shall mean any work of authorship, including
//    the original version of the Work and any modifications or additions
//    to that Work or Derivative Works thereof, that is intentionally
//    submitted to Licensor for inclusion in the Work by the copyright owner
//    or by an individual or Legal Entity authorized to submit on behalf of
//    the copyright owner. For the purposes of this definition, "submitted"
//    means any form of electronic, verbal, or written communication sent
//    to the Licensor or its representatives, including but not limited to
//    communication on electronic mailing lists, source code control systems,
//    and issue tracking systems that are managed by, or on behalf of, the
//    Licensor for the purpose of discussing and improving the Work, but
//    excluding communication that is conspicuously marked or otherwise
//    designated in writing by the copyright owner as "Not a Contribution."
//
//    "Contributor" shall mean Licensor and any individual or Legal Entity
//    on behalf of whom a Contribution has been received by Licensor and
//    subsequently incorporated within the Work.
//
// 2. Grant of Copyright License. Subject to the terms and conditions of
//    this License, each Contributor hereby grants to You a perpetual,
//    worldwide, non-exclusive, no-charge, royalty-free, irrevocable
//    copyright license to reproduce, prepare Derivative Works of,
//    publicly display, publicly perform, sublicense, and distribute the
//    Work and such Derivative Works in Source or Object form.
//
// 3. Grant of Patent License. Subject to the terms and conditions of
//    this License, each Contributor hereby grants to You a perpetual,
//    worldwide, non-exclusive, no-charge, royalty-free, irrevocable
//    (except as stated in this section) patent license to make, have made,
//    use, offer to sell, sell, import, and otherwise transfer the Work,
//    where such license applies only to those patent claims licensable
//    by such Contributor that are necessarily infringed by their
//    Contribution(s) alone or by combination of their Contribution(s)
//    with the Work to which such Contribution(s) was submitted. If You
//    institute patent litigation against any entity (including a
//    cross-claim or counterclaim in a lawsuit) alleging that the Work
//    or a Contribution incorporated within the Work constitutes direct
//    or contributory patent infringement, then any patent licenses
//    granted to You under this License for that Work shall terminate
//    as of the date such litigation is filed.
//
// 4. Redistribution. You may reproduce and distribute copies of the
//    Work or Derivative Works thereof in any medium, with or without
//    modifications, and in Source or Object form, provided that You
//    meet the following conditions:
//
//    (a) You must give any other recipients of the Work or
//        Derivative Works a copy of this License; and
//
//    (b) You must cause any modified files to carry prominent notices
//        stating that You changed the files; and
//
//    (c) You must retain, in the Source form of any Derivative Works
//        that You distribute, all copyright, patent, trademark, and
//        attribution notices from the Source form of the Work,
//        excluding those notices that do not pertain to any part of
//        the Derivative Works; and
//
//    (d) If the Work includes a "NOTICE" text file as part of its
//        distribution, then any Derivative Works that You distribute must
//        include a readable copy of the attribution notices contained
//        within such NOTICE file, excluding those notices that do not
//        pertain to any part of the Derivative Works, in at least one
//        of the following places: within a NOTICE text file distributed
//        as part of the Derivative Works; within the Source form or
//        documentation, if provided along with the Derivative Works; or,
//        within a display generated by the Derivative Works, if and
//        wherever such third-party notices normally appear. The contents
//        of the NOTICE file are for informational purposes only and
//        do not modify the License. You may add Your own attribution
//        notices within Derivative Works that You distribute, alongside
//        or as an addendum to the NOTICE text from the Work, provided
//        that such additional attribution notices cannot be construed
//        as modifying the License.
//
//    You may add Your own copyright statement to Your modifications and
//    may provide additional or different license terms and conditions
//    for use, reproduction, or distribution of Your modifications, or
//    for any such Derivative Works as a whole, provided Your use,
//    reproduction, and distribution of the Work otherwise complies with
//    the conditions stated in this License.
//
// 5. Submission of Contributions. Unless You explicitly state otherwise,
//    any Contribution intentionally submitted for inclusion in the Work
//    by You to the Licensor shall be under the terms and conditions of
//    this License, without any additional terms or conditions.
//    Notwithstanding the above, nothing herein shall supersede or modify
//    the terms of any separate license agreement you may have executed
//    with Licensor regarding such Contributions.
//
// 6. Trademarks. This License does not grant permission to use the trade
//    names, trademarks, service marks, or product names of the Licensor,
//    except as required for reasonable and customary use in describing the
//    origin of the Work and reproducing the content of the NOTICE file.
//
// 7. Disclaimer of Warranty. Unless required by applicable law or
//    agreed to in writing, Licensor provides the Work (and each
//    Contributor provides its Contributions) on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
//    implied, including, without limitation, any warranties or conditions
//    of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
//    PARTICULAR PURPOSE. You are solely responsible for determining the
//    appropriateness of using or redistributing the Work and assume any
//    risks associated with Your exercise of permissions under this License.
//
// 8. Limitation of Liability. In no event and under no legal theory,
//    whether in tort (including negligence), contract, or otherwise,
//    unless required by applicable law (such as deliberate and grossly
//    negligent acts) or agreed to in writing, shall any Contributor be
//    liable to You for damages, including any direct, indirect, special,
//    incidental, or consequential damages of any character arising as a
//    result of this License or out of the use or inability to use the
//    Work (including but not limited to damages for loss of goodwill,
//    work stoppage, computer failure or malfunction, or any and all
//    other commercial damages or losses), even if such Contributor
//    has been advised of the possibility of such damages.
//
// 9. Accepting Warranty or Additional Liability. While redistributing
//    the Work or Derivative Works thereof, You may choose to offer,
//    and charge a fee for, acceptance of support, warranty, indemnity,
//    or other liability obligations and/or rights consistent with this
//    License. However, in accepting such obligations, You may act only
//    on Your own behalf and on Your sole responsibility, not on behalf
//    of any other Contributor, and only if You agree to indemnify,
//    defend, and hold each Contributor harmless for any liability
//    incurred by, or claims asserted against, such Contributor by reason
//    of your accepting any such warranty or additional liability.
//
// END OF TERMS AND CONDITIONS
//
// Copyright 2013-2018 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package patternmatcher

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWildcardMatches(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"*"})
	if !match {
		t.Errorf("failed to get a wildcard match, got %v", match)
	}
}

// A simple pattern match should return true.
func TestPatternMatches(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"*.go"})
	if !match {
		t.Errorf("failed to get a match, got %v", match)
	}
}

// An exclusion followed by an inclusion should return true.
func TestExclusionPatternMatchesPatternBefore(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"!fileutils.go", "*.go"})
	if !match {
		t.Errorf("failed to get true match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderExclusions(t *testing.T) {
	match, _ := Matches("docs/README.md", []string{"docs", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWithSlashExclusions(t *testing.T) {
	match, _ := Matches("docs/README.md", []string{"docs/", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWildcardExclusions(t *testing.T) {
	match, _ := Matches("docs/README.md", []string{"docs/*", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A pattern followed by an exclusion should return false.
func TestExclusionPatternMatchesPatternAfter(t *testing.T) {
	match, _ := Matches("fileutils.go", []string{"*.go", "!fileutils.go"})
	if match {
		t.Errorf("failed to get false match on exclusion pattern, got %v", match)
	}
}

// A filename evaluating to . should return false.
func TestExclusionPatternMatchesWholeDirectory(t *testing.T) {
	match, _ := Matches(".", []string{"*.go"})
	if match {
		t.Errorf("failed to get false match on ., got %v", match)
	}
}

// A single ! pattern should return an error.
func TestSingleExclamationError(t *testing.T) {
	_, err := Matches("fileutils.go", []string{"!"})
	if err == nil {
		t.Errorf("failed to get an error for a single exclamation point, got %v", err)
	}
}

// Matches with no patterns
func TestMatchesWithNoPatterns(t *testing.T) {
	matches, err := Matches("/any/path/there", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if matches {
		t.Fatalf("Should not have match anything")
	}
}

// Matches with malformed patterns
func TestMatchesWithMalformedPatterns(t *testing.T) {
	matches, err := Matches("/any/path/there", []string{"["})
	if err == nil {
		t.Fatal("Should have failed because of a malformed syntax in the pattern")
	}
	if matches {
		t.Fatalf("Should not have match anything")
	}
}

type matchesTestCase struct {
	pattern string
	text    string
	pass    bool
}

type multiPatternTestCase struct {
	patterns []string
	text     string
	pass     bool
}

func TestMatches(t *testing.T) {
	tests := []matchesTestCase{
		{"**", "file", true},
		{"**", "file/", true},
		{"**/", "file", true}, // weird one
		{"**/", "file/", true},
		{"**", "/", true},
		{"**/", "/", true},
		{"**", "dir/file", true},
		{"**/", "dir/file", true},
		{"**", "dir/file/", true},
		{"**/", "dir/file/", true},
		{"**/**", "dir/file", true},
		{"**/**", "dir/file/", true},
		{"dir/**", "dir/file", true},
		{"dir/**", "dir/file/", true},
		{"dir/**", "dir/dir2/file", true},
		{"dir/**", "dir/dir2/file/", true},
		{"**/dir", "dir", true},
		{"**/dir", "dir/file", true},
		{"**/dir2/*", "dir/dir2/file", true},
		{"**/dir2/*", "dir/dir2/file/", true},
		{"**/dir2/**", "dir/dir2/dir3/file", true},
		{"**/dir2/**", "dir/dir2/dir3/file/", true},
		{"**file", "file", true},
		{"**file", "dir/file", true},
		{"**/file", "dir/file", true},
		{"**file", "dir/dir/file", true},
		{"**/file", "dir/dir/file", true},
		{"**/file*", "dir/dir/file", true},
		{"**/file*", "dir/dir/file.txt", true},
		{"**/file*txt", "dir/dir/file.txt", true},
		{"**/file*.txt", "dir/dir/file.txt", true},
		{"**/file*.txt*", "dir/dir/file.txt", true},
		{"**/**/*.txt", "dir/dir/file.txt", true},
		{"**/**/*.txt2", "dir/dir/file.txt", false},
		{"**/*.txt", "file.txt", true},
		{"**/**/*.txt", "file.txt", true},
		{"a**/*.txt", "a/file.txt", true},
		{"a**/*.txt", "a/dir/file.txt", true},
		{"a**/*.txt", "a/dir/dir/file.txt", true},
		{"a/*.txt", "a/dir/file.txt", false},
		{"a/*.txt", "a/file.txt", true},
		{"a/*.txt**", "a/file.txt", true},
		{"a[b-d]e", "ae", false},
		{"a[b-d]e", "ace", true},
		{"a[b-d]e", "aae", false},
		{"a[^b-d]e", "aze", true},
		{".*", ".foo", true},
		{".*", "foo", false},
		{"abc.def", "abcdef", false},
		{"abc.def", "abc.def", true},
		{"abc.def", "abcZdef", false},
		{"abc?def", "abcZdef", true},
		{"abc?def", "abcdef", false},
		{"a\\\\", "a\\", true},
		{"**/foo/bar", "foo/bar", true},
		{"**/foo/bar", "dir/foo/bar", true},
		{"**/foo/bar", "dir/dir2/foo/bar", true},
		{"abc/**", "abc", false},
		{"abc/**", "abc/def", true},
		{"abc/**", "abc/def/ghi", true},
		{"**/.foo", ".foo", true},
		{"**/.foo", "bar.foo", false},
		{"a(b)c/def", "a(b)c/def", true},
		{"a(b)c/def", "a(b)c/xyz", false},
		{"a.|)$(}+{bc", "a.|)$(}+{bc", true},
		{"dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", true},
		{"dist/*.whl", "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", true},
	}
	multiPatternTests := []multiPatternTestCase{
		{[]string{"**", "!util/docker/web"}, "util/docker/web/foo", false},
		{[]string{"**", "!util/docker/web", "util/docker/web/foo"}, "util/docker/web/foo", true},
		{[]string{"**", "!dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl"}, "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", false},
		{[]string{"**", "!dist/*.whl"}, "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", false},
	}

	if runtime.GOOS != "windows" {
		tests = append(tests, []matchesTestCase{
			{"a\\*b", "a*b", true},
		}...)
	}

	t.Run("MatchesOrParentMatches", func(t *testing.T) {
		for _, test := range tests {
			pm, err := New([]string{test.pattern})
			if err != nil {
				t.Fatalf("%v (pattern=%q, text=%q)", err, test.pattern, test.text)
			}
			res, _ := pm.MatchesOrParentMatches(test.text)
			if test.pass != res {
				t.Fatalf("%v (pattern=%q, text=%q)", err, test.pattern, test.text)
			}
		}

		for _, test := range multiPatternTests {
			pm, err := New(test.patterns)
			if err != nil {
				t.Fatalf("%v (patterns=%q, text=%q)", err, test.patterns, test.text)
			}
			res, _ := pm.MatchesOrParentMatches(test.text)
			if test.pass != res {
				t.Errorf("expected: %v, got: %v (patterns=%q, text=%q)", test.pass, res, test.patterns, test.text)
			}
		}
	})

	t.Run("MatchesUsingParentResult", func(t *testing.T) {
		for _, test := range tests {
			pm, err := New([]string{test.pattern})
			if err != nil {
				t.Fatalf("%v (pattern=%q, text=%q)", err, test.pattern, test.text)
			}

			parentPath := filepath.Dir(filepath.FromSlash(test.text))
			parentPathDirs := strings.Split(parentPath, string(os.PathSeparator))

			parentMatched := false
			if parentPath != "." {
				for i := range parentPathDirs {
					parentMatched, _ = pm.MatchesUsingParentResult(strings.Join(parentPathDirs[:i+1], "/"), parentMatched)
				}
			}

			res, _ := pm.MatchesUsingParentResult(test.text, parentMatched)
			if test.pass != res {
				t.Errorf("expected: %v, got: %v (pattern=%q, text=%q)", test.pass, res, test.pattern, test.text)
			}
		}
	})

	t.Run("MatchesUsingParentResults", func(t *testing.T) {
		check := func(pm *PatternMatcher, text string, pass bool, desc string) {
			parentPath := filepath.Dir(filepath.FromSlash(text))
			parentPathDirs := strings.Split(parentPath, string(os.PathSeparator))

			parentMatchInfo := MatchInfo{}
			if parentPath != "." {
				for i := range parentPathDirs {
					_, parentMatchInfo, _ = pm.MatchesUsingParentResults(strings.Join(parentPathDirs[:i+1], "/"), parentMatchInfo)
				}
			}

			res, _, _ := pm.MatchesUsingParentResults(text, parentMatchInfo)
			if pass != res {
				t.Errorf("expected: %v, got: %v %s", pass, res, desc)
			}
		}

		for _, test := range tests {
			desc := fmt.Sprintf("(pattern=%q text=%q)", test.pattern, test.text)
			pm, err := New([]string{test.pattern})
			if err != nil {
				t.Fatal(err, desc)
			}

			check(pm, test.text, test.pass, desc)
		}

		for _, test := range multiPatternTests {
			desc := fmt.Sprintf("pattern=%q text=%q", test.patterns, test.text)
			pm, err := New(test.patterns)
			if err != nil {
				t.Fatal(err, desc)
			}

			check(pm, test.text, test.pass, desc)
		}
	})

	t.Run("MatchesUsingParentResultsNoContext", func(t *testing.T) {
		check := func(pm *PatternMatcher, text string, pass bool, desc string) {
			res, _, _ := pm.MatchesUsingParentResults(text, MatchInfo{})
			if pass != res {
				t.Errorf("expected: %v, got: %v %s", pass, res, desc)
			}
		}

		for _, test := range tests {
			desc := fmt.Sprintf("(pattern=%q text=%q)", test.pattern, test.text)
			pm, err := New([]string{test.pattern})
			if err != nil {
				t.Fatal(err, desc)
			}

			check(pm, test.text, test.pass, desc)
		}

		for _, test := range multiPatternTests {
			desc := fmt.Sprintf("(pattern=%q text=%q)", test.patterns, test.text)
			pm, err := New(test.patterns)
			if err != nil {
				t.Fatal(err, desc)
			}

			check(pm, test.text, test.pass, desc)
		}
	})
}

func TestCleanPatterns(t *testing.T) {
	patterns := []string{"docs", "config"}
	pm, err := New(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	cleaned := pm.Patterns()
	if len(cleaned) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(cleaned))
	}
}

func TestCleanPatternsStripEmptyPatterns(t *testing.T) {
	patterns := []string{"docs", "config", ""}
	pm, err := New(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	cleaned := pm.Patterns()
	if len(cleaned) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(cleaned))
	}
}

func TestCleanPatternsExceptionFlag(t *testing.T) {
	patterns := []string{"docs", "!docs/README.md"}
	pm, err := New(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	if !pm.Exclusions() {
		t.Errorf("expected exceptions to be true, got %v", pm.Exclusions())
	}
}

func TestCleanPatternsLeadingSpaceTrimmed(t *testing.T) {
	patterns := []string{"docs", "  !docs/README.md"}
	pm, err := New(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	if !pm.Exclusions() {
		t.Errorf("expected exceptions to be true, got %v", pm.Exclusions())
	}
}

func TestCleanPatternsTrailingSpaceTrimmed(t *testing.T) {
	patterns := []string{"docs", "!docs/README.md  "}
	pm, err := New(patterns)
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	if !pm.Exclusions() {
		t.Errorf("expected exceptions to be true, got %v", pm.Exclusions())
	}
}

func TestCleanPatternsErrorSingleException(t *testing.T) {
	patterns := []string{"!"}
	_, err := New(patterns)
	if err == nil {
		t.Errorf("expected error on single exclamation point, got %v", err)
	}
}

// These matchTests are stolen from go's filepath Match tests.
type matchTest struct {
	pattern, s string
	match      bool
	err        error
}

var matchTests = []matchTest{
	{"abc", "abc", true, nil},
	{"*", "abc", true, nil},
	{"*c", "abc", true, nil},
	{"a*", "a", true, nil},
	{"a*", "abc", true, nil},
	{"a*", "ab/c", true, nil},
	{"a*/b", "abc/b", true, nil},
	{"a*/b", "a/c/b", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
	{"ab[c]", "abc", true, nil},
	{"ab[b-d]", "abc", true, nil},
	{"ab[e-g]", "abc", false, nil},
	{"ab[^c]", "abc", false, nil},
	{"ab[^b-d]", "abc", false, nil},
	{"ab[^e-g]", "abc", true, nil},
	{"a\\*b", "a*b", true, nil},
	{"a\\*b", "ab", false, nil},
	{"a?b", "a☺b", true, nil},
	{"a[^a]b", "a☺b", true, nil},
	{"a???b", "a☺b", false, nil},
	{"a[^a][^a][^a]b", "a☺b", false, nil},
	{"[a-ζ]*", "α", true, nil},
	{"*[a-ζ]", "A", false, nil},
	{"a?b", "a/b", false, nil},
	{"a*b", "a/b", false, nil},
	{"[\\]a]", "]", true, nil},
	{"[\\-]", "-", true, nil},
	{"[x\\-]", "x", true, nil},
	{"[x\\-]", "-", true, nil},
	{"[x\\-]", "z", false, nil},
	{"[\\-x]", "x", true, nil},
	{"[\\-x]", "-", true, nil},
	{"[\\-x]", "a", false, nil},
	{"[]a]", "]", false, filepath.ErrBadPattern},
	{"[-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "x", false, filepath.ErrBadPattern},
	{"[x-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "z", false, filepath.ErrBadPattern},
	{"[-x]", "x", false, filepath.ErrBadPattern},
	{"[-x]", "-", false, filepath.ErrBadPattern},
	{"[-x]", "a", false, filepath.ErrBadPattern},
	{"\\", "a", false, filepath.ErrBadPattern},
	{"[a-b-c]", "a", false, filepath.ErrBadPattern},
	{"[", "a", false, filepath.ErrBadPattern},
	{"[^", "a", false, filepath.ErrBadPattern},
	{"[^bc", "a", false, filepath.ErrBadPattern},
	{"a[", "a", false, filepath.ErrBadPattern}, // was nil but IMO its wrong
	{"a[", "ab", false, filepath.ErrBadPattern},
	{"*x", "xxx", true, nil},
}

func errp(e error) string {
	if e == nil {
		return "<nil>"
	}
	return e.Error()
}

// TestMatch tests our version of filepath.Match, called Matches.
func TestMatch(t *testing.T) {
	for _, tt := range matchTests {
		pattern := tt.pattern
		s := tt.s
		if runtime.GOOS == "windows" {
			if strings.Contains(pattern, "\\") {
				// no escape allowed on windows.
				continue
			}
			pattern = filepath.Clean(pattern)
			s = filepath.Clean(s)
		}
		ok, err := Matches(s, []string{pattern})
		if ok != tt.match || err != tt.err {
			t.Fatalf("Match(%#q, %#q) = %v, %q want %v, %q", pattern, s, ok, errp(err), tt.match, errp(tt.err))
		}
	}
}

type compileTestCase struct {
	pattern               string
	matchType             matchType
	compiledRegexp        string
	windowsCompiledRegexp string
}

var compileTests = []compileTestCase{
	{"*", regexpMatch, `^[^/]*$`, `^[^\\]*$`},
	{"file*", regexpMatch, `^file[^/]*$`, `^file[^\\]*$`},
	{"*file", regexpMatch, `^[^/]*file$`, `^[^\\]*file$`},
	{"a*/b", regexpMatch, `^a[^/]*/b$`, `^a[^\\]*\\b$`},
	{"**", suffixMatch, "", ""},
	{"**/**", regexpMatch, `^(.*/)?.*$`, `^(.*\\)?.*$`},
	{"dir/**", prefixMatch, "", ""},
	{"**/dir", suffixMatch, "", ""},
	{"**/dir2/*", regexpMatch, `^(.*/)?dir2/[^/]*$`, `^(.*\\)?dir2\\[^\\]*$`},
	{"**/dir2/**", regexpMatch, `^(.*/)?dir2/.*$`, `^(.*\\)?dir2\\.*$`},
	{"**file", suffixMatch, "", ""},
	{"**/file*txt", regexpMatch, `^(.*/)?file[^/]*txt$`, `^(.*\\)?file[^\\]*txt$`},
	{"**/**/*.txt", regexpMatch, `^(.*/)?(.*/)?[^/]*\.txt$`, `^(.*\\)?(.*\\)?[^\\]*\.txt$`},
	{"a[b-d]e", regexpMatch, `^a[b-d]e$`, `^a[b-d]e$`},
	{".*", regexpMatch, `^\.[^/]*$`, `^\.[^\\]*$`},
	{"abc.def", exactMatch, "", ""},
	{"abc?def", regexpMatch, `^abc[^/]def$`, `^abc[^\\]def$`},
	{"**/foo/bar", suffixMatch, "", ""},
	{"a(b)c/def", exactMatch, "", ""},
	{"a.|)$(}+{bc", exactMatch, "", ""},
	{"dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", exactMatch, "", ""},
}

// TestCompile confirms that "compile" assigns the correct match type to a
// variety of test case patterns. If the match type is regexp, it also confirms
// that the compiled regexp matches the expected regexp.
func TestCompile(t *testing.T) {
	t.Run("slash", testCompile("/"))
	t.Run("backslash", testCompile(`\`))
}

func testCompile(sl string) func(*testing.T) {
	return func(t *testing.T) {
		for _, tt := range compileTests {
			// Avoid NewPatternMatcher, which has platform-specific behavior
			pm := &PatternMatcher{
				patterns: make([]*Pattern, 1),
			}
			pattern := path.Clean(tt.pattern)
			if sl != "/" {
				pattern = strings.ReplaceAll(pattern, "/", sl)
			}
			newp := &Pattern{}
			newp.cleanedPattern = pattern
			newp.dirs = strings.Split(pattern, sl)
			pm.patterns[0] = newp

			if err := pm.patterns[0].compile(sl); err != nil {
				t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
			}
			if pm.patterns[0].matchType != tt.matchType {
				t.Errorf("pattern %q: matchType = %v, want %v", pattern, pm.patterns[0].matchType, tt.matchType)
				continue
			}
			if tt.matchType == regexpMatch {
				if sl == `\` {
					if pm.patterns[0].regexp.String() != tt.windowsCompiledRegexp {
						t.Errorf("pattern %q: regexp = %s, want %s", pattern, pm.patterns[0].regexp, tt.windowsCompiledRegexp)
					}
				} else if pm.patterns[0].regexp.String() != tt.compiledRegexp {
					t.Errorf("pattern %q: regexp = %s, want %s", pattern, pm.patterns[0].regexp, tt.compiledRegexp)
				}
			}
		}
	}
}
