package table

import "io"

type IconProvider interface {
	Get(name string) string
	Write(name string, writer io.Writer)
}
