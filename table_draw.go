package table

import (
	"bytes"
	"io"
	"net/url"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

/******************************************
 * View Methods (Write to Buffers)
 *******************************************/

func (widget *Table) Draw(params *url.URL, buffer io.Writer) error {

	query := params.Query()

	// Parse and clamp the focus column to a valid index, since it comes from untrusted query input
	focusColumn, _ := strconv.Atoi(query.Get("focus"))
	if (focusColumn < 0) || (focusColumn >= len(widget.Form.Children)) {
		focusColumn = 0
	}

	// Try to ADD a row
	if query.Get("add") == "true" {
		return widget.drawTable(null.Int{}, true, focusColumn, buffer)
	}

	// Try to EDIT a row
	if edit := query.Get("edit"); edit != "" {
		if editIndex, err := strconv.Atoi(edit); err == nil {
			return widget.drawTable(null.NewInt(editIndex), false, focusColumn, buffer)
		}
	}

	// Otherwise, just draw the table (view only)
	return widget.drawTable(null.Int{}, false, focusColumn, buffer)
}

// DrawView returns a VIEW ONLY representation of the table
func (widget *Table) DrawView(buffer io.Writer) error {
	return widget.drawTable(null.Int{}, false, 0, buffer)
}

// DrawAdd returns the table with a row for adding a new record
func (widget *Table) DrawAdd(buffer io.Writer) error {
	return widget.drawTable(null.Int{}, true, 0, buffer)
}

// DrawEdit returns the table with a single editable row
func (widget *Table) DrawEdit(index int, buffer io.Writer) error {
	return widget.drawTable(null.NewInt(index), false, 0, buffer)
}

/******************************************
 * String Wrappers for View Methods
 ******************************************/

// DrawViewString returns a string representation of the table (VIEW ONLY)
func (widget *Table) DrawViewString() (string, error) {

	const location = "table.Widget.DrawViewString"

	var buffer bytes.Buffer

	if err := widget.DrawView(&buffer); err != nil {
		return "", derp.Wrap(err, location, "Rendering table")
	}

	return buffer.String(), nil

}

// DrawAddString returns a string representation of the table with a row for adding a new record
func (widget *Table) DrawAddString() (string, error) {

	const location = "table.Widget.DrawAddString"

	var buffer bytes.Buffer

	if err := widget.DrawAdd(&buffer); err != nil {
		return "", derp.Wrap(err, location, "Rendering table")
	}

	return buffer.String(), nil

}

// DrawEditString returns a string representation of the table with a single editable row
func (widget *Table) DrawEditString(index int) (string, error) {

	const location = "table.Widget.DrawEditString"

	var buffer bytes.Buffer

	if err := widget.DrawEdit(index, &buffer); err != nil {
		return "", derp.Wrap(err, location, "Rendering table")
	}

	return buffer.String(), nil
}

/******************************************
 * Draw Methods (these do the actual work of rendering the table)
 ******************************************/

// draw writes this table to the provided io.Writer
func (widget *Table) drawTable(editRow null.Int, addRow bool, focusColumn int, buffer io.Writer) error {

	const location = "table.Widget.drawTable"

	// Collect metadata
	tableElement, err := widget.getTableElement()

	if err != nil {
		return derp.Wrap(err, location, "Getting table element")
	}

	rowElement := tableElement.Items
	tableSchema := schema.New(tableElement)
	rowSchema := schema.New(rowElement)

	tableValue, _ := widget.Schema.Get(widget.Object, widget.Path)
	tableLength := convert.SliceLength(tableValue)

	// Compute the effective permissions for THIS render as locals, so the table data's
	// min/max bounds never mutate the caller's widget.
	canAdd := widget.CanAdd
	canEdit := widget.CanEdit
	canDelete := widget.CanDelete

	// Only allow ADDs if the table is smaller than the maximum value
	if (tableElement.MaxLength > 0) && (tableLength >= tableElement.MaxLength) {
		canAdd = false
	}

	// Only allow DELETEs if the table is larger than the minimum value
	if tableLength <= tableElement.MinLength {
		canDelete = false
	}

	//
	// Verify Permissions Here
	//

	if canAdd && addRow {

		// If adding is allowed and requested, then set the editable row to a new row at the end of the table
		editRow.Set(tableLength)

	} else if canEdit && editRow.IsPresent() {

		// If editing is allowed and requested, then bounds check the editRow
		// If the editRow is out of bounds, then use view-only mode
		if (editRow.Int() < 0) || (editRow.Int() >= tableLength) {
			editRow.Unset()
		}

	} else {

		// All other cases, use view-only mode
		editRow.Unset()
	}

	// Begin rendering the widget
	b := html.New()

	// Wrapper
	if editRow.IsPresent() {
		b.Form("", "").
			Class("grid").
			Data("hx-post", widget.getURL("edit", editRow.Int(), 0)).
			Data("hx-target", "this").
			Data("hx-swap", "outerHTML").
			Data("hx-push-url", "false")

	} else {

		b.Div().
			Class("grid").
			Data("hx-target", "this").
			Data("hx-swap", "outerHTML").
			Data("hx-push-url", "false")
	}

	// Table
	b.Table().
		Class("grid")

	// Header row
	b.TR().Class("grid-header")
	for _, field := range widget.Form.Children {
		td := b.TD().Class("grid-cell") // nolint:scopeguard
		if width, ok := field.Options["column-width"]; ok {
			td.Style("width", convert.String(width))
		}
		b.Div().InnerText(field.Label).Close()
		b.Close() // TD
	}
	b.TD().Class("grid-cell", "grid-controls").Close()
	b.Close() // TR

	// Data rows
	for rowIndex := 0; rowIndex < tableLength; rowIndex++ {

		rowValue, err := tableSchema.Get(tableValue, strconv.Itoa(rowIndex))

		if err != nil {
			return derp.Wrap(err, location, "Getting row data", tableSchema, tableValue, rowIndex, tableLength)
		}

		if canEdit && editRow.IsPresent() && (editRow.Int() == rowIndex) {

			if err := widget.drawEditRow(&rowSchema, rowValue, canEdit, focusColumn, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Drawing row (edit)", widget.Path, rowIndex)
			}

		} else {

			if err := widget.drawViewRow(&rowSchema, rowIndex, rowValue, canEdit, canDelete, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Drawing row (view)", widget.Path, rowIndex)
			}
		}
	}

	// If we're not editing an existing row, then let users add a new row
	if canAdd {
		if addRow {
			if err := widget.drawAddRow(&rowSchema, canAdd, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Drawing row (add)", widget.Path, tableLength)
			}
		} else {
			b.Close() // TABLE
			b.Div()
			b.Button().
				Type("button").
				Class("link").
				Data("hx-get", widget.getURL("add", tableLength, 0)).
				InnerHTML(widget.Icons.Get("plus") + " Add a Row")
			b.Close() // Button
			b.Close() // Div
		}
	}

	b.CloseAll()

	if _, err := buffer.Write(b.Bytes()); err != nil {
		return derp.Wrap(err, location, "Writing table HTML to buffer", widget.Path)
	}

	return nil

}

// focusField returns a copy of the form element with its "focus" option enabled.
// It clones the Options map so the shared Form definition is never mutated during rendering.
func focusField(field form.Element) form.Element {

	options := make(mapof.Any, len(field.Options)+1)
	for key, value := range field.Options {
		options[key] = value
	}
	options["focus"] = true

	field.Options = options
	return field
}

func (widget *Table) drawAddRow(rowSchema *schema.Schema, canAdd bool, b *html.Builder) error {

	const location = "table.Widget.drawAddRow"

	// Paranoid double-check
	if !canAdd {
		return nil
	}

	b.TR().Class("grid-row", "grid-editable")

	width := "width:calc(100% / " + strconv.Itoa(len(widget.Form.Children)) + ")"
	f := form.New(*rowSchema, *widget.Form)

	for column, field := range widget.Form.Children {
		b.TD().Class("grid-cell", "grid-editable").Style(width)

		// Focus the first column when adding a new row
		if column == 0 {
			field = focusField(field)
		}

		if err := field.Edit(&f, widget.LookupProvider, nil, b.SubTree()); err != nil {
			return derp.Wrap(err, location, "Rendering field", field)
		}
		b.Close() // TD
	}

	b.TD().Class("grid-cell", "grid-editable", "grid-controls")
	b.Button().Type("submit").Class("text-green").InnerHTML(widget.Icons.Get("save")).Close()
	b.Space()
	b.Button().Type("button").Data("hx-get", widget.TargetURL).InnerHTML(widget.Icons.Get("cancel")).Close()
	b.Close() // TD

	b.Close() // TR
	return nil
}

func (widget *Table) drawEditRow(rowSchema *schema.Schema, rowValue any, canEdit bool, focusColumn int, b *html.Builder) error {

	const location = "table.Widget.drawEditRow"

	// Paranoid double-check
	if !canEdit {
		return derp.Internal(location, "Editing is not allowed.  THIS SHOULD NEVER HAPPEN")
	}

	b.TR().Class("grid-row", "grid-editable")

	width := "width:calc(100% / " + strconv.Itoa(len(widget.Form.Children)) + ")"
	f := form.New(*rowSchema, *widget.Form)

	for index, field := range widget.Form.Children {

		b.TD().Class("grid-cell", "grid-editable").Style(width)

		// Focus the requested column when editing.  An out-of-range focusColumn
		// simply matches no column, so no field is focused (and nothing panics).
		if index == focusColumn {
			field = focusField(field)
		}

		if err := field.Edit(&f, widget.LookupProvider, rowValue, b.SubTree()); err != nil {
			return derp.Wrap(err, location, "Rendering field", field)
		}
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

func (widget *Table) drawViewRow(rowSchema *schema.Schema, rowIndex int, rowValue any, canEdit bool, canDelete bool, b *html.Builder) error {

	const location = "table.Widget.drawViewRow"

	b.TR().Class("grid-row", "hover-trigger")

	width := "width:calc(100% / " + strconv.Itoa(len(widget.Form.Children)) + ")"
	f := form.New(*rowSchema, *widget.Form)

	for colIndex, field := range widget.Form.Children {

		cell := b.TD().Class("grid-cell").Style(width) // nolint:scopeguard

		if canEdit {
			cell.Data("hx-get", widget.getURL("edit", rowIndex, colIndex)).Data("hx-trigger", "click")
		}

		if err := field.View(&f, widget.LookupProvider, rowValue, b.SubTree()); err != nil {
			return derp.Wrap(err, location, "Rendering field", field)
		}

		b.Close() // TD
	}

	b.TD().Class("grid-cell", "grid-controls")

	if canEdit {
		b.Button().
			Type("button").
			Data("hx-get", widget.getURL("edit", rowIndex, 0)).
			InnerHTML(widget.Icons.Get("edit")).
			Close()
	}

	if canDelete {
		b.Space()
		b.Button().
			Type("button").
			Data("hx-post", widget.getURL("delete", rowIndex, 0)).
			Data("hx-confirm", "Are you sure you want to delete this row?").
			InnerHTML(widget.Icons.Get("delete")).Close()
	}

	b.Close() // TD
	b.Close() // TR

	return nil
}
