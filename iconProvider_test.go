package table

import "io"

type testIconProvider struct{}

func (t testIconProvider) Get(name string) string {
	return name
}

func (t testIconProvider) Write(name string, writer io.Writer) {
	writer.Write([]byte(name))
}
