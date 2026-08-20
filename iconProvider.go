package table

import "io"

// IconProvider generates the HTML for the icons used by the table's controls.
type IconProvider interface {

	// Get returns the HTML for the named icon.
	Get(name string) string

	// Write writes the HTML for the named icon to the provided writer.
	Write(name string, writer io.Writer)
}
