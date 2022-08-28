package table

import (
	"bytes"
	"io"
	"reflect"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/null"
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

	// Optional Fields
	LookupProvider form.LookupProvider // Optional dependency to provide lookup data for fields
	CanAdd         bool                // If TRUE, then users can add new rows to the table
	CanEdit        bool                // If TRUE, then users can edit existing rows in the table
	CanDelete      bool                // If TRUE, then users can delete existing rows in the table
}

// New returns a fully initialiized Table widget (with all required fields)
func New(schema *schema.Schema, form *form.Element, object any, path string, editable bool, targetURL string) Table {
	return Table{
		Schema:    schema,
		Form:      form,
		Object:    object,
		Path:      path,
		TargetURL: targetURL,
		CanAdd:    editable,
		CanEdit:   editable,
		CanDelete: editable,
	}
}

/********************************
 * View Methods (Write to Buffers)
 ********************************/

// DrawView returns a VIEW ONLY representation of the table
func (widget *Table) DrawView(buffer io.Writer) error {
	return widget.drawTable(null.Int{}, false, buffer)
}

// DrawAdd returns the table with a row for adding a new record
func (widget *Table) DrawAdd(buffer io.Writer) error {
	return widget.drawTable(null.Int{}, true, buffer)
}

// DrawEdit returns the table with a single editable row
func (widget *Table) DrawEdit(index int, buffer io.Writer) error {
	return widget.drawTable(null.NewInt(index), false, buffer)
}

/********************************
 * String Wrappers for View Methods
 ********************************/

// DrawViewString returns a string representation of the table (VIEW ONLY)
func (widget *Table) DrawViewString() (string, error) {
	var buffer bytes.Buffer

	if err := widget.DrawView(&buffer); err != nil {
		return "", derp.Wrap(err, "table.Widget.ViewString", "Error rendering table")
	}

	return buffer.String(), nil

}

// DrawAddString returns a string representation of the table with a row for adding a new record
func (widget *Table) DrawAddString() (string, error) {
	var buffer bytes.Buffer

	if err := widget.DrawAdd(&buffer); err != nil {
		return "", derp.Wrap(err, "table.Widget.ViewString", "Error rendering table")
	}

	return buffer.String(), nil

}

// DrawEditString returns a string representation of the table with a single editable row
func (widget *Table) DrawEditString(index int) (string, error) {
	var buffer bytes.Buffer

	if err := widget.DrawEdit(index, &buffer); err != nil {
		return "", derp.Wrap(err, "table.Widget.ViewString", "Error rendering table")
	}

	return buffer.String(), nil
}

/********************************
 * Draw Methods (these do the actual work of rendering the table)
 ********************************/

// draw writes this table to the provided io.Writer
func (widget *Table) drawTable(editRow null.Int, addRow bool, buffer io.Writer) error {

	const location = "table.Widget.drawTable"

	// Try to locate (and validate) that we have a usable schema for a table
	value, tableElement, err := widget.Schema.Get(widget.Object, widget.Path)

	if err != nil {
		return derp.Wrap(err, location, "Failed to locate table schema", widget.Path)
	}

	// Validate that the table schema is a slicee/array
	if _, ok := tableElement.(schema.Array); !ok {
		return derp.NewInternalError(location, "Table schema must be an array", widget.Path, tableElement)
	}

	// Collect metadata
	arraySchema := schema.New(tableElement)
	valueOf := reflect.ValueOf(value)
	length := convert.SliceLength(value)

	//
	// VERIFY PERMISSIONS HERE
	//

	if widget.CanAdd && addRow {

		// If adding is allowed and requested, then set the editable row to a new row at the end of the table
		editRow.Set(length)

	} else if widget.CanEdit && editRow.IsPresent() {

		// If editing is allowed and requested, then bounds check the editRow
		// If the editRow is out of bounds, then use view-only mode
		if (editRow.Int() < 0) || (editRow.Int() >= length) {
			editRow.Unset()
		}

	} else {

		// All other cases, use view-only mode
		editRow.Unset()
	}

	// Begin rendering the table
	b := html.New()

	if editRow.IsPresent() {
		b.Form("", "").
			Data("hx-post", widget.getURL("edit", editRow.Int())).
			Data("hx-target", "this").
			Data("hx-swap", "outerHTML").
			Data("hx-push_url", "false")

		b.Table()
	} else {
		b.Table().
			Data("hx-target", "this").
			Data("hx-swap", "outerHTML").
			Data("hx-push_url", "false")
	}

	// Header row
	b.TR()
	for _, field := range widget.Form.Children {
		b.TH().InnerHTML(field.Label).Close()
	}
	b.TH().Close()
	b.Close() // TR

	// Data rows
	for rowIndex := 0; rowIndex < length; rowIndex++ {

		// Get the data for this row
		rowData, rowElement, err := arraySchema.Get(valueOf, strconv.Itoa(rowIndex))

		if err != nil {
			return derp.Wrap(err, location, "Failed to locate row schema", widget.Path, rowIndex)
		}

		rowSchema := schema.New(rowElement)

		if widget.CanEdit && editRow.IsPresent() && (editRow.Int() == rowIndex) {

			if err := widget.drawEditRow(&rowSchema, rowData, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Failed to draw edit row", widget.Path, rowIndex)
			}

		} else {

			if err := widget.drawViewRow(&rowSchema, rowIndex, rowData, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Failed to draw row", widget.Path, rowIndex)
			}
		}
	}

	// If we're not editing an existing row, then let users add a new row
	if widget.CanAdd && (editRow.Int() == length) {
		widget.drawAddRow(widget.Schema, b.SubTree())
	}

	b.CloseAll()

	buffer.Write(b.Bytes())
	return nil

}

func (widget Table) drawAddRow(rowSchema *schema.Schema, b *html.Builder) {

	// Paranoid double-check
	if !widget.CanAdd {
		return
	}

	b.TR().Class("add")

	for _, field := range widget.Form.Children {
		b.TD()
		field.WriteHTML(rowSchema, nil, nil, b.SubTree())
		b.Close() // .cell
	}

	b.TD().Class("align-right", "text-lg")
	b.I("ti", "ti-circle-plus").Role("button").Close()

	b.Close() // .cell
	// b.Close() // .row
	b.Close() // form
}

func (widget Table) drawEditRow(rowSchema *schema.Schema, rowData any, b *html.Builder) error {

	// Paranoid double-check
	if !widget.CanEdit {
		return derp.NewInternalError("table.Widget.drawEditRow", "Editing is not allowed.  THIS SHOULD NEVER HAPPEN")
	}

	b.TR().Class("edit")

	for _, field := range widget.Form.Children {
		b.TD()
		field.WriteHTML(rowSchema, widget.LookupProvider, rowData, b.SubTree())
		b.Close() // .cell
	}

	// Write actions column
	b.TD().Class("align-right", "text-lg")
	b.I("ti", "ti-circle-check", "text-green").
		Role("button").
		Type("submit").
		Close()

	b.Space()

	b.I("ti", "ti-x", "text-red").
		Role("button").
		Data("hx-get", widget.TargetURL).
		Close()

	b.Close() // TR

	return nil
}

func (widget Table) drawViewRow(rowSchema *schema.Schema, rowIndex int, rowData any, b *html.Builder) error {

	row := b.TR().Class("hover-trigger")

	if widget.CanEdit {
		row.Role("button").Data("hx-get", widget.getURL("edit", rowIndex)).Data("hx-trigger", "click")
	}

	for _, field := range widget.Form.Children {

		b.TD()
		field.WriteHTML(rowSchema, widget.LookupProvider, rowData, b.SubTree())
		b.Close() // TD
	}

	b.Div().Role("cell").Class("align-right", "text-lg")

	if widget.CanEdit {
		b.I("ti", "ti-pencil").
			Data("hx-get", widget.getURL("edit", rowIndex)).
			Close()
	}

	if widget.CanDelete {
		b.Space()
		b.I("ti", "ti-trash").
			Role("button").
			Data("hx-confirm", "Are you sure you want to delete this row?").
			Data("hx-post", widget.getURL("delete", rowIndex)).
			Close()
	}

	b.Close() // .cell
	b.Close() // .row

	return nil
}

/********************************
 * Update/Delete Methods
 ********************************/

// DoEdit applies a dataset to the requested row in the table
func (widget *Table) DoEdit(data maps.Map, editIndex int) error {

	const location = "table.Widget.DoEdit"

	rowData, _, err := widget.Schema.Get(data, strconv.Itoa(editIndex))

	if err != nil {
		return derp.Wrap(err, location, "Failed to locate row schema", widget.Path, editIndex)
	}

	length := convert.SliceLength(rowData)

	// Bounds checking
	if editIndex < 0 {
		return derp.NewInternalError(location, "Edit index out of range (negative index not allowed)", widget.Path, editIndex)
	}

	if editIndex > length {
		// NOTE: allow editIndex to equal length to add a new row
		return derp.NewInternalError(location, "Edit index out of range (too large)", widget.Path, editIndex)
	}

	// Try to add/edit the row in the data table
	for _, field := range widget.Form.AllElements() {
		path := list.ByDot(widget.Path, strconv.Itoa(editIndex), field.Path).String()

		if err := widget.Schema.Set(widget.Object, path, data[field.Path]); err != nil {
			return derp.Wrap(err, location, "Error setting value in table", widget.Path, path, data)
		}
	}

	return nil
}

// DoDelete removes the requested row from the table
func (widget *Table) DoDelete(data maps.Map, deleteIndex int) error {

	const location = "table.Widget.DoEdit"

	path := list.ByDot(widget.Path, strconv.Itoa(deleteIndex)).String()

	if err := widget.Schema.Remove(widget.Object, path); err != nil {
		return derp.Wrap(err, location, "Error removing value from table")
	}

	return nil
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
	return widget.TargetURL + "?" + action + "=" + convert.String(row)
}
