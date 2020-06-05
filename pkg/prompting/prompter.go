package prompting

// Prompter is the interface to which types supporting prompting must adhere.
// Implementations are not required to be safe for concurrent usage.
type Prompter interface {
	// Message should print a message to the user, returning an error if this is
	// not possible.
	Message(string) error
	// Prompt should print a prompt to the user, returning the user's response
	// or an error if this is not possible.
	Prompt(string) (string, error)
}
