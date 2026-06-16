package table

import "io"

// IconProvider generates the HTML for the icons used by the table's controls.
type IconProvider interface {
	Get(name string) string
	Write(name string, writer io.Writer)
}
