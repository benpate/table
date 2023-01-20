package table

import (
	"net/url"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/maps"
)

/********************************
 * Update/Delete Methods
 ********************************/

func (widget *Table) Do(queryParams *url.URL, data maps.Map) error {

	if edit := queryParams.Query().Get("edit"); edit != "" {

		if editIndex, err := strconv.Atoi(edit); err == nil {
			if err := widget.DoEdit(data, editIndex); err != nil {
				return derp.Wrap(err, "table.Widget.Do", "Failed to edit row", widget.Path, editIndex)
			}
		}
	} else if delete := queryParams.Query().Get("delete"); delete != "" {

		if deleteIndex, err := strconv.Atoi(delete); err == nil {
			if err := widget.DoDelete(deleteIndex); err != nil {
				return derp.Wrap(err, "table.Widget.Do", "Failed to delete row", widget.Path, deleteIndex)
			}
		}
	}

	// Success!
	return nil
}

// DoEdit applies a dataset to the requested row in the table
func (widget *Table) DoEdit(data maps.Map, editIndex int) error {

	const location = "table.Widget.DoEdit"

	// Locate the table data and validate the length of the existing array
	tableData, err := widget.Schema.Get(widget.Object, widget.Path)

	if err != nil {
		return derp.Wrap(err, location, "Failed to locate row schema", widget.Path, editIndex)
	}

	length := convert.SliceLength(tableData)

	switch {

	// Cannot be negative index
	case editIndex < 0:
		return derp.NewInternalError(location, "Edit index out of range (negative index not allowed)", widget.Path, editIndex)

	// Cannot be greater than length (but equal to length is okay because it means "add a new row")
	case editIndex > length:
		return derp.NewInternalError(location, "Edit index out of range (too large)", data, widget.Path, tableData, length, editIndex)

	// Verify permission to add
	case editIndex == length:
		if !widget.CanAdd {
			return derp.NewInternalError(location, "Cannot add new row", widget.Path, editIndex)
		}

	// Verify permission to edit
	default:
		if !widget.CanEdit {
			return derp.NewInternalError(location, "Cannot edit row", widget.Path, editIndex)
		}
	}

	// Try to add/edit the row in the data table
	for _, field := range widget.Form.AllElements() {
		path := list.ByDot(widget.Path, strconv.Itoa(editIndex), field.Path)
		if err := widget.Schema.Set(widget.Object, path.String(), data[field.Path]); err != nil {
			return derp.Wrap(err, location, "Error setting value in table", path.String(), data)
		}
	}

	// Success!
	return nil
}

// DoDelete removes the requested row from the table
func (widget *Table) DoDelete(deleteIndex int) error {

	const location = "table.Widget.DoEdit"

	if !widget.CanDelete {
		return derp.NewBadRequestError(location, "Deleting is not allowed", widget.Path)
	}

	path := list.ByDot(widget.Path, strconv.Itoa(deleteIndex)).String()

	if ok := widget.Schema.Remove(widget.Object, path); !ok {
		return derp.NewInternalError(location, "Error removing value from table")
	}

	return nil
}
