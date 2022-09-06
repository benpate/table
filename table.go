package table

import (
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

/********************************
 * Configuration Methods
 ********************************/

// AllowAdd modifies the table to allow adding new rows.
func (widget *Table) AllowAdd() *Table {
	widget.CanAdd = true
	return widget
}

// AllowEdit modifies the table to allow editing existing rows.
func (widget *Table) AllowEdit() *Table {
	widget.CanEdit = true
	return widget
}

// AllowDelete modifies the table to allow deleting existing rows.
func (widget *Table) AllowDelete() *Table {
	widget.CanDelete = true
	return widget
}

// AllowAll modifies the table to allow all write actions (Add, Edit, Delete).
func (widget *Table) AllowAll() *Table {
	widget.CanAdd = true
	widget.CanEdit = true
	widget.CanDelete = true
	return widget
}

// AllowNone modifies the table to disallow all write actions.
func (widget *Table) AllowNone() *Table {
	widget.CanAdd = false
	widget.CanEdit = false
	widget.CanDelete = false
	return widget
}

// UseLookupProvider modifies the table to use the given lookup provider.
func (widget *Table) UseLookupProvider(lookupProvider form.LookupProvider) *Table {
	widget.LookupProvider = lookupProvider
	return widget
}

/********************************
 * Other Convenience Methods
 ********************************/

// getURL returns a safe URL to use in callbacks.
func (widget *Table) getURL(action string, row int) string {

	switch action {
	case "add":
		return widget.TargetURL + "?add=true"
	case "edit":
		return widget.TargetURL + "?edit=" + convert.String(row)
	case "delete":
		return widget.TargetURL + "?delete=" + convert.String(row)
	default:
		return widget.TargetURL
	}
}

func (widget *Table) getTableElement() (schema.Array, error) {

	if widget.Schema == nil {
		return schema.Array{}, derp.New(derp.CodeInternalError, "table.Widget.getTableElement", "Schema is nil", widget.Path)
	}

	element, err := widget.Schema.GetElement(widget.Path)

	if err != nil {
		return schema.Array{}, derp.Wrap(err, "table.Widget.getTableElement", "Failed to get table element", widget.Path)
	}

	arrayElement, ok := element.(schema.Array)

	if !ok {
		return schema.Array{}, derp.NewBadRequestError("table.Widget.getTableElement", "Table element is not an array", widget.Path)
	}

	return arrayElement, nil
}

/*
func (widget *Table) getRowElement() schema.Element {
	return widget.getTableElement().Items
}
*/
