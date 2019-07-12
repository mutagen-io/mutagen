package core

// DefaultVCSIgnores is the default set of ignores to use when ignoring VCS
// directories.
var DefaultVCSIgnores = []string{
	".git/",
	".svn/",
	".hg/",
	".bzr/",
	"_darcs/",
}
