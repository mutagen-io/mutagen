package filesystem

// windowsLongPathTestingLength is the path length to use when testing for long
// path support on Windows. Any path of 248 or more characters is sufficient to
// be rejected under Windows' default path handling (according to the Go
// standard library comments), so we pick a length to use that's a bit larger.
// Instead of creating a chain of directories exceeding this length, our tests
// generally just create a single component with this length in order to ensure
// the limit is exceeded. However, the maximum path length of any single
// component is 255 characters, so we have to stay beneath that.
const windowsLongPathTestingLength = 252
