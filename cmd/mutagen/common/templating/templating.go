package templating

import (
	"bytes"
	"encoding/json"
	"text/template"
)

// jsonify is the built-in JSON encoder that's made available to templates.
func jsonify(value any) (string, error) {
	// Create a buffer to store the output.
	buffer := &bytes.Buffer{}

	// Create and configure a JSON encoder.
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)

	// Marshal the value. Note that every call to Encode also adds a newline
	// after the value's JSON representation.
	if err := encoder.Encode(value); err != nil {
		return "", err
	}

	// Convert the encoded JSON to a string.
	return buffer.String(), nil
}

// builtins are the builtin functions supported in output templates.
var builtins = template.FuncMap{
	"json": jsonify,
	// TODO: Figure out what other functions we want to include here, if any.
}
