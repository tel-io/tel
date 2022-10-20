package health

// Version is the current release version of the health instrumentation.
func Version() string {
	return "0.32.0"
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
func SemVersion() string {
	return "semver:" + Version()
}
