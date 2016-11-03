package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	flag "github.com/ogier/pflag"

	"github.com/pkg/errors"
)

type FlagSet struct {
	*flag.FlagSet
	usage                 string
	validPositionalCounts []int
}

func NewFlagSet(name, usage string, validPositionalCounts []int) *FlagSet {
	// Create the flag set.
	flags := flag.NewFlagSet(name, flag.ContinueOnError)

	// Disable the flag set's built-in help and error printing.
	flags.SetOutput(ioutil.Discard)

	// Create our wrapper.
	return &FlagSet{flags, usage, validPositionalCounts}
}

func (f *FlagSet) Parse(arguments []string) error {
	panic("the ParseOrDie method should be used instead")
}

func (f *FlagSet) ParseOrDie(arguments []string) []string {
	// Invoke the underlying parser, handling cases of help requests and parsing
	// errors.
	if err := f.FlagSet.Parse(arguments); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(os.Stdout, f.usage)
			Die(false)
		} else {
			Error(err)
			fmt.Fprint(os.Stderr, f.usage)
			Die(true)
		}
	}

	// Watch for cases where an invalid number of arguments are specified.
	nArg := f.NArg()
	correctNArg := false
	if len(f.validPositionalCounts) == 0 && nArg == 0 {
		correctNArg = true
	} else {
		for _, c := range f.validPositionalCounts {
			if c == -1 || nArg == c {
				correctNArg = true
				break
			}
		}
	}
	if !correctNArg {
		Error(errors.New("invalid number of positional arguments"))
		fmt.Fprint(os.Stderr, f.usage)
		Die(true)
	}

	// Return positional arguments.
	return f.Args()
}
