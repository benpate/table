package table

import "io"

// testIconProvider is a stub IconProvider that echoes each icon name back, so
// tests can assert on the name instead of a blob of icon markup.
type testIconProvider struct{}

// Get returns the icon name verbatim.
func (t testIconProvider) Get(name string) string {
	return name
}

// Write writes the icon name verbatim.
func (t testIconProvider) Write(name string, writer io.Writer) {
	_, _ = writer.Write([]byte(name)) // errors are irrelevant in this test helper
}
