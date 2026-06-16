package table

import (
	"net/url"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
)

/******************************************
 * Update/Delete Methods
 ******************************************/

// Do applies an edit or delete action to the table's data, selecting the action
// from the "edit" and "delete" query parameters.
func (widget *Table) Do(queryParams *url.URL, data map[string]any) error {

	const location = "table.Widget.Do"

	// If this is an edit request, then apply the data to the requested row
	if edit := queryParams.Query().Get("edit"); edit != "" {

		if editIndex, err := strconv.Atoi(edit); err == nil {
			if err := widget.DoEdit(data, editIndex); err != nil {
				return derp.Wrap(err, location, "Editing row", widget.Path, editIndex)
			}
		}

		return nil
	}

	// If this is a delete request, then remove the requested row
	if deleteParam := queryParams.Query().Get("delete"); deleteParam != "" {

		if deleteIndex, err := strconv.Atoi(deleteParam); err == nil {
			if err := widget.DoDelete(deleteIndex); err != nil {
				return derp.Wrap(err, location, "Deleting row", widget.Path, deleteIndex)
			}
		}

		return nil
	}

	// Nothing to do here
	return nil
}

// DoEdit applies a dataset to the requested row in the table
func (widget *Table) DoEdit(data map[string]any, editIndex int) error {

	const location = "table.Widget.DoEdit"

	// Locate the table data and validate the length of the existing array
	tableData, err := widget.Schema.Get(widget.Object, widget.Path)

	if err != nil {
		return derp.Wrap(err, location, "Locating row schema", widget.Path, editIndex)
	}

	length := convert.SliceLength(tableData) //nolint:scopeguard

	switch {

	// Cannot be negative index
	case editIndex < 0:
		return derp.Internal(location, "Edit index out of range (negative index not allowed)", widget.Path, editIndex)

	// Cannot be greater than length (but equal to length is okay because it means "add a new row")
	case editIndex > length:
		return derp.Internal(location, "Edit index out of range (too large)", data, widget.Path, tableData, length, editIndex)

	// Verify permission to add
	case editIndex == length:
		if !widget.CanAdd {
			return derp.Internal(location, "Cannot add new row", widget.Path, editIndex)
		}

	// Verify permission to edit
	default:
		if !widget.CanEdit {
			return derp.Internal(location, "Cannot edit row", widget.Path, editIndex)
		}
	}

	// Try to add/edit the row in the data table
	for _, field := range widget.Form.AllElements() {
		path := list.ByDot(widget.Path, strconv.Itoa(editIndex), field.Path)
		if err := widget.Schema.Set(widget.Object, path.String(), data[field.Path]); err != nil {
			return derp.Wrap(err, location, "Setting value in table", path.String(), data)
		}
	}

	// Success!
	return nil
}

// DoDelete removes the requested row from the table
func (widget *Table) DoDelete(deleteIndex int) error {

	const location = "table.Widget.DoDelete"

	if !widget.CanDelete {
		return derp.BadRequest(location, "Deleting is not allowed", widget.Path)
	}

	path := list.ByDot(widget.Path, strconv.Itoa(deleteIndex)).String()

	if ok := widget.Schema.Remove(widget.Object, path); !ok {
		return derp.Internal(location, "Removing value from table")
	}

	return nil
}
