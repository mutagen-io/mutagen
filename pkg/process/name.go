package process

func ExecutableName(base, goos string) string {
	if goos == "windows" {
		return base + ".exe"
	}
	return base
}
