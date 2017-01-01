package sync

type Problem struct {
	Path  string
	Error string
}

func newProblem(path string, err error) Problem {
	return Problem{
		Path:  path,
		Error: err.Error(),
	}
}
