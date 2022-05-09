package templating

import (
	"errors"
	"fmt"
	"os"
	"text/template"
	"unicode/utf8"

	"github.com/spf13/pflag"
)

// TemplateFlags stores command line formatting flags and provides for their
// registration and handling.
type TemplateFlags struct {
	// template stores the value of the --template flag.
	template string
	// templateFile stores the value of the --template-file flag.
	templateFile string
}

// Register registers the flags into the specified flag set.
func (f *TemplateFlags) Register(flags *pflag.FlagSet) {
	flags.StringVar(&f.template, "template", "", "Specify an output template")
	flags.StringVar(&f.templateFile, "template-file", "", "Specify a file containing an output template")
}

// LoadTemplate loads the template specified by the flags. If no template has
// been specified, then it returns nil with no error.
func (f *TemplateFlags) LoadTemplate() (*template.Template, error) {
	// Figure out if there's a template to be processed. If not, then no valid
	// template has been specified and we can just return.
	var literal string
	if f.template != "" {
		literal = f.template
	} else if f.templateFile != "" {
		if l, err := os.ReadFile(f.templateFile); err != nil {
			return nil, fmt.Errorf("unable to load template: %w", err)
		} else if !utf8.Valid(l) {
			return nil, errors.New("template file is not UTF-8 encoded")
		} else {
			literal = string(l)
		}
	} else {
		return nil, nil
	}

	// Create the template and register built-in functions.
	result := template.New("")
	result.Funcs(builtins)

	// Parse the template literal.
	return result.Parse(literal)
}
