package table

import "io"

type testIconProvider struct{}

func (t testIconProvider) Get(name string) string {
	return name
}

func (t testIconProvider) Write(name string, writer io.Writer) {
	// nolint:errcheck // ignore errors in test code
	writer.Write([]byte(name))
}
