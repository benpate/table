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
	tableElement, err := widget.Schema.GetElement(widget.Path)

	if err != nil {
		return derp.Wrap(err, location, "Failed to locate table schema", widget.Path)
	}

	// Validate that the table schema is a slicee/array
	if _, ok := tableElement.(schema.Array); !ok {
		return derp.NewInternalError(location, "Table schema must be an array", widget.Path, tableElement)
	}

	value, _ := widget.Schema.Get(widget.Object, widget.Path)

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

	columnCount := len(widget.Form.Children)
	columnWidth := convert.String(100/columnCount) + "%"

	// Begin rendering the widget
	b := html.New()

	// Wrapper
	if editRow.IsPresent() {
		b.Form("", "").
			Class("grid").
			Data("hx-post", widget.getURL("edit", editRow.Int())).
			Data("hx-target", "this").
			Data("hx-swap", "outerHTML").
			Data("hx-push_url", "false")

	} else {

		b.Div().
			Class("grid").
			Data("hx-target", "this").
			Data("hx-swap", "outerHTML").
			Data("hx-push_url", "false")
	}

	// Table
	b.Table().
		Class("grid")

	// Header row
	b.TR().Class("grid-header")
	for _, field := range widget.Form.Children {
		b.TD().Class("grid-cell").Style("width:" + columnWidth)
		b.Div().InnerHTML(field.Label).Close()
		b.Close() // TD
	}
	b.TD().Class("grid-cell", "grid-controls").Close()
	b.Close() // TR

	rowElement, err := arraySchema.GetElement("0")

	if err != nil {
		return derp.Wrap(err, location, "Failed to locate row schema", widget.Path)
	}

	rowSchema := schema.New(rowElement)

	// Data rows
	for rowIndex := 0; rowIndex < length; rowIndex++ {

		// Get the data for this row

		rowData, _ := arraySchema.Get(valueOf, strconv.Itoa(rowIndex))

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
	if widget.CanAdd {
		if addRow {
			widget.drawAddRow(&rowSchema, b.SubTree())
		} else {
			b.Close() // TABLE
			b.Div()
			b.Button().
				Type("button").
				Data("hx-get", widget.getURL("add", length)).
				InnerHTML(widget.Icons.Get("plus") + " Add a Row")
			b.Close() // Button
			b.Close() // Div
		}
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

	b.TR().Class("grid-row", "grid-editable")

	for _, field := range widget.Form.Children {
		b.TD().Class("grid-cell", "grid-editable")
		field.Edit(rowSchema, widget.LookupProvider, nil, b.SubTree())
		b.Close() // TD
	}

	b.TD().Class("grid-cell", "grid-editable", "grid-controls")
	b.Button().Type("submit").Class("text-green").InnerHTML(widget.Icons.Get("save")).Close()
	b.Space()
	b.Button().Type("button").Data("hx-get", widget.TargetURL).InnerHTML(widget.Icons.Get("cancel")).Close()
	b.Close() // TD

	b.Close() // TR
}

func (widget Table) drawEditRow(rowSchema *schema.Schema, rowData any, b *html.Builder) error {

	// Paranoid double-check
	if !widget.CanEdit {
		return derp.NewInternalError("table.Widget.drawEditRow", "Editing is not allowed.  THIS SHOULD NEVER HAPPEN")
	}

	b.TR().Class("grid-row", "grid-editable")

	for _, field := range widget.Form.Children {
		b.TD().Class("grid-cell", "grid-editable")
		field.Edit(rowSchema, widget.LookupProvider, rowData, b.SubTree())
		b.Close() // TD
	}

	// Write actions column
	b.TD().Class("grid-cell", "grid-editable", "grid-controls")
	b.Button().Type("submit").Class("text-green").InnerHTML(widget.Icons.Get("save")).Close()
	b.Space()
	b.Button().Type("button").Data("hx-get", widget.TargetURL).InnerHTML(widget.Icons.Get("cancel")).Close()
	b.Close() // TR

	return nil
}

func (widget Table) drawViewRow(rowSchema *schema.Schema, rowIndex int, rowData any, b *html.Builder) error {

	b.TR().Class("grid-row", "hover-trigger")

	for _, field := range widget.Form.Children {

		cell := b.TD().Class("grid-cell")

		if widget.CanEdit {
			cell.Data("hx-get", widget.getURL("edit", rowIndex)).Data("hx-trigger", "click")
		}

		field.View(rowSchema, widget.LookupProvider, rowData, b.SubTree())
		b.Close() // TD
	}

	b.TD().Class("grid-cell", "grid-controls")

	if widget.CanEdit {
		b.Button().
			Type("button").
			Data("hx-get", widget.getURL("edit", rowIndex)).
			InnerHTML(widget.Icons.Get("edit")).
			Close()
	}

	if widget.CanDelete {
		b.Space()
		b.Button().
			Type("button").
			Data("hx-post", widget.getURL("delete", rowIndex)).
			Data("hx-confirm", "Are you sure you want to delete this row?").
			InnerHTML(widget.Icons.Get("delete")).Close()
	}

	b.Close() // TD
	b.Close() // TR

	return nil
}

/********************************
 * Update/Delete Methods
 ********************************/

// DoEdit applies a dataset to the requested row in the table
func (widget *Table) DoEdit(data maps.Map, editIndex int) error {

	const location = "table.Widget.DoEdit"

	rowData, err := widget.Schema.Get(data, strconv.Itoa(editIndex))

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

	if action == "view" {
		return widget.TargetURL
	}

	return widget.TargetURL + "?" + action + "=" + convert.String(row)
}
