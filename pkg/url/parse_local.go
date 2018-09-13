package url

// parseLocal parses a local URL. It simply assumes the URL refers to a local
// path.
func parseLocal(raw string) (*URL, error) {
	return &URL{
		Protocol: Protocol_Local,
		Path:     raw,
	}, nil
}
