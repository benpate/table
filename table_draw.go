package table

import (
	"bytes"
	"io"
	"net/url"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

/********************************
 * View Methods (Write to Buffers)
 ********************************/

func (widget *Table) Draw(params *url.URL, buffer io.Writer) error {

	// Try to ADD a row
	if params.Query().Get("add") == "true" {
		return widget.DrawAdd(buffer)
	}

	// Try to EDIT a row
	if edit := params.Query().Get("edit"); edit != "" {
		if editIndex, err := strconv.Atoi(edit); err == nil {
			return widget.DrawEdit(editIndex, buffer)
		}
	}

	// Otherwise, just draw the table
	return widget.DrawView(buffer)
}

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

	// Collect metadata
	tableElement, err := widget.getTableElement()

	if err != nil {
		return derp.Wrap(err, location, "Error getting table element")
	}

	rowElement := tableElement.Items
	tableSchema := schema.New(tableElement)
	rowSchema := schema.New(rowElement)

	tableValue, _ := widget.Schema.Get(widget.Object, widget.Path)
	tableLength := convert.SliceLength(tableValue)

	// Only allow ADDs if the table is smaller than the maximum value
	if (tableElement.MaxLength > 0) && (tableLength >= tableElement.MaxLength) {
		widget.CanAdd = false
	}

	// Only allow DELETEs if the table is larger than the minimum value
	if tableLength <= tableElement.MinLength {
		widget.CanDelete = false
	}

	//
	// Verify Permissions Here
	//

	if widget.CanAdd && addRow {

		// If adding is allowed and requested, then set the editable row to a new row at the end of the table
		editRow.Set(tableLength)

	} else if widget.CanEdit && editRow.IsPresent() {

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
		td := b.TD().Class("grid-cell")
		if width, ok := field.Options["column-width"]; ok {
			td.Style("width", convert.String(width))
		}
		b.Div().InnerHTML(field.Label).Close()
		b.Close() // TD
	}
	b.TD().Class("grid-cell", "grid-controls").Close()
	b.Close() // TR

	// Data rows
	for rowIndex := 0; rowIndex < tableLength; rowIndex++ {

		rowValue, err := tableSchema.Get(tableValue, strconv.Itoa(rowIndex))

		if err != nil {
			return derp.Wrap(err, location, "Error getting row data", tableSchema, tableValue, rowIndex, tableLength)
		}

		if widget.CanEdit && editRow.IsPresent() && (editRow.Int() == rowIndex) {

			if err := widget.drawEditRow(&rowSchema, rowValue, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Failed to draw row (edit)", widget.Path, rowIndex)
			}

		} else {

			if err := widget.drawViewRow(&rowSchema, rowIndex, rowValue, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Failed to draw row (view)", widget.Path, rowIndex)
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
				Class("link").
				Data("hx-get", widget.getURL("add", tableLength)).
				InnerHTML(widget.Icons.Get("plus") + " Add a Row")
			b.Close() // Button
			b.Close() // Div
		}
	}

	b.CloseAll()

	buffer.Write(b.Bytes())
	return nil

}

func (widget *Table) drawAddRow(rowSchema *schema.Schema, b *html.Builder) {

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

func (widget *Table) drawEditRow(rowSchema *schema.Schema, rowValue any, b *html.Builder) error {

	// Paranoid double-check
	if !widget.CanEdit {
		return derp.NewInternalError("table.Widget.drawEditRow", "Editing is not allowed.  THIS SHOULD NEVER HAPPEN")
	}

	b.TR().Class("grid-row", "grid-editable")

	for _, field := range widget.Form.Children {
		b.TD().Class("grid-cell", "grid-editable")
		field.Edit(rowSchema, widget.LookupProvider, rowValue, b.SubTree())
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

func (widget *Table) drawViewRow(rowSchema *schema.Schema, rowIndex int, rowValue any, b *html.Builder) error {

	b.TR().Class("grid-row", "hover-trigger")

	for _, field := range widget.Form.Children {

		cell := b.TD().Class("grid-cell")

		if widget.CanEdit {
			cell.Data("hx-get", widget.getURL("edit", rowIndex)).Data("hx-trigger", "click")
		}

		field.View(rowSchema, widget.LookupProvider, rowValue, b.SubTree())
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
