// Package table renders an editable HTML table (grid) widget whose columns are
// defined by a form.Element and whose data is described by a rosetta schema.
package table

import (
	"net/url"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/schema"
)

// Table defines all of the properties of a table widget
type Table struct {
	// Required Fields
	Schema    *schema.Schema // Data schema for the data in this table
	Form      *form.Element  // Form (UI Schema) that defines the columns being displayed
	Object    any            // Object containing the table data
	Path      string         // Path to the data in the object
	TargetURL string         // URL to send the form data to
	Icons     IconProvider   // IconProvider generates HTML for icons

	// Optional Fields
	LookupProvider form.LookupProvider // Optional dependency to provide lookup data for fields
	CanAdd         bool                // If TRUE, then users can add new rows to the table
	CanEdit        bool                // If TRUE, then users can edit existing rows in the table
	CanDelete      bool                // If TRUE, then users can delete existing rows in the table
}

// New returns a fully initialiized Table widget (with all required fields)
func New(schema *schema.Schema, form *form.Element, object any, path string, iconProvider IconProvider, targetURL string) Table {
	return Table{
		Schema:    schema,
		Form:      form,
		Object:    object,
		Path:      path,
		TargetURL: targetURL,
		Icons:     iconProvider,
		CanAdd:    true,
		CanEdit:   true,
		CanDelete: true,
	}
}

/******************************************
 * Configuration Methods
 ******************************************/

// These builders take a value receiver and return a modified copy, so they can be
// chained directly off New (e.g. New(...).AllowNone()) without escaping to the heap.
// Because they do not mutate in place, callers must use the returned value.

// AllowAdd returns a copy of the table that allows adding new rows.
func (widget Table) AllowAdd() Table {
	widget.CanAdd = true
	return widget
}

// AllowEdit returns a copy of the table that allows editing existing rows.
func (widget Table) AllowEdit() Table {
	widget.CanEdit = true
	return widget
}

// AllowDelete returns a copy of the table that allows deleting existing rows.
func (widget Table) AllowDelete() Table {
	widget.CanDelete = true
	return widget
}

// AllowAll returns a copy of the table that allows all write actions (Add, Edit, Delete).
func (widget Table) AllowAll() Table {
	widget.CanAdd = true
	widget.CanEdit = true
	widget.CanDelete = true
	return widget
}

// AllowNone returns a copy of the table that disallows all write actions.
func (widget Table) AllowNone() Table {
	widget.CanAdd = false
	widget.CanEdit = false
	widget.CanDelete = false
	return widget
}

// UseLookupProvider returns a copy of the table that uses the given lookup provider.
func (widget Table) UseLookupProvider(lookupProvider form.LookupProvider) Table {
	widget.LookupProvider = lookupProvider
	return widget
}

/*******************************************
 * Other Convenience Methods
 ******************************************/

// getURL returns a safe URL to use in callbacks, merging the action's query
// parameters into any query string the TargetURL already has.
func (widget *Table) getURL(action string, row int, col int) string {

	parsed, err := url.Parse(widget.TargetURL)

	// If the TargetURL can't be parsed, fall back to returning it unchanged
	if err != nil {
		return widget.TargetURL
	}

	query := parsed.Query()

	switch action {
	case "add":
		query.Set("add", "true")
	case "edit":
		query.Set("edit", convert.String(row))
		query.Set("focus", convert.String(col))
	case "delete":
		query.Set("delete", convert.String(row))
	default:
		return widget.TargetURL
	}

	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (widget *Table) getTableElement() (schema.Array, error) {

	const location = "table.Widget.getTableElement"

	if widget.Schema == nil {
		return schema.Array{}, derp.Internal(location, "Schema is nil", widget.Path)
	}

	element, ok := widget.Schema.GetElement(widget.Path)

	if !ok {
		return schema.Array{}, derp.Internal(location, "Getting table element", widget.Schema, widget.Path)
	}

	arrayElement, ok := element.(schema.Array)

	if !ok {
		return schema.Array{}, derp.Internal(location, "Table element is not an array", widget.Path)
	}

	return arrayElement, nil
}
