package table

import (
	"bytes"
	"testing"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Draw() - Query Param Router
 ******************************************/

func TestDraw_View(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x"), &buffer)

	require.NoError(t, err)
	result := buffer.String()
	assert.Contains(t, result, `<div class="grid"`) // view mode wraps in a <div>
	assert.Contains(t, result, "John Connor")
	assert.NotContains(t, result, "<form")
}

func TestDraw_Add(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?add=true"), &buffer)

	require.NoError(t, err)
	result := buffer.String()
	assert.Contains(t, result, "<form")              // add mode wraps in a <form>
	assert.Contains(t, result, `<input name="name"`) // editable input row
}

func TestDraw_Edit(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?edit=0"), &buffer)

	require.NoError(t, err)
	result := buffer.String()
	assert.Contains(t, result, "<form")
	assert.Contains(t, result, `value="John Connor"`) // existing value loaded into the input
}

func TestDraw_EditNotNumeric(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	// A non-numeric "edit" value falls through to a normal (view-only) render
	err := table.Draw(mustURL(t, "http://x?edit=abc"), &buffer)

	require.NoError(t, err)
	result := buffer.String()
	assert.Contains(t, result, `<div class="grid"`)
	assert.NotContains(t, result, "<form")
}

func TestDraw_FocusParam(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?focus=1"), &buffer)

	require.NoError(t, err)
	assert.Equal(t, 1, table.focusColumn) // focus query param is parsed into focusColumn
}

// A focus column beyond the last column is untrusted input and must not panic.
func TestDraw_FocusTooLarge(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?edit=0&focus=999"), &buffer)

	require.NoError(t, err)
	assert.Equal(t, 0, table.focusColumn) // clamped back to a valid column
}

// A negative focus column must not panic.
func TestDraw_FocusNegative(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?add=true&focus=-1"), &buffer)

	require.NoError(t, err)
	assert.Equal(t, 0, table.focusColumn) // clamped back to a valid column
}

/******************************************
 * DrawViewString()
 ******************************************/

func TestDrawViewString(t *testing.T) {

	table := newTestTable()

	result, err := table.DrawViewString()

	require.NoError(t, err)

	// Column headers
	assert.Contains(t, result, "<div>Name</div>")
	assert.Contains(t, result, "<div>Age</div>")

	// Row data
	assert.Contains(t, result, "John Connor")
	assert.Contains(t, result, "Sarah Connor")

	// Default permissions render edit, delete, and "Add a Row" controls
	assert.Contains(t, result, "edit")           // edit icon
	assert.Contains(t, result, "delete")         // delete icon
	assert.Contains(t, result, "plus Add a Row") // add button
}

func TestDrawViewString_AllowNone(t *testing.T) {

	table := newTestTable()
	table.AllowNone()

	result, err := table.DrawViewString()

	require.NoError(t, err)

	// Data still renders...
	assert.Contains(t, result, "John Connor")

	// ...but no write controls are present
	assert.NotContains(t, result, "Add a Row")
	assert.NotContains(t, result, "hx-confirm") // delete buttons carry an hx-confirm
}

func TestDrawViewString_MaxLengthDisablesAdd(t *testing.T) {

	// A table that is already at MaxLength cannot add new rows
	table := newTestTable()
	table.Schema = pointerTo(schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"data": schema.Array{
					MaxLength: 2,
					Items:     schema.Object{Properties: schema.ElementMap{"name": schema.String{}, "age": schema.Integer{}}},
				},
			},
		},
	})

	result, err := table.DrawViewString()

	require.NoError(t, err)
	assert.False(t, table.CanAdd) // drawTable flips CanAdd off
	assert.NotContains(t, result, "Add a Row")
}

func TestDrawViewString_MinLengthDisablesDelete(t *testing.T) {

	// A table that is at (or below) MinLength cannot delete rows
	table := newTestTable()
	db := table.Object.(*testDatabase)
	db.Data = sliceof.Object[mapof.Any]{mapof.Any{"name": "Last One", "age": 99}} // single row, MinLength == 1

	result, err := table.DrawViewString()

	require.NoError(t, err)
	assert.False(t, table.CanDelete) // drawTable flips CanDelete off
	assert.NotContains(t, result, "hx-confirm")
}

func TestDrawViewString_Error(t *testing.T) {

	table := newTestTable()
	table.Schema = nil // getTableElement will fail

	result, err := table.DrawViewString()

	require.Error(t, err)
	assert.Empty(t, result)
}

/******************************************
 * DrawAddString()
 ******************************************/

func TestDrawAddString(t *testing.T) {

	table := newTestTable()

	result, err := table.DrawAddString()

	require.NoError(t, err)
	assert.Contains(t, result, "<form")
	assert.Contains(t, result, `<input name="name"`)
	assert.Contains(t, result, "save")   // save icon
	assert.Contains(t, result, "cancel") // cancel icon
}

func TestDrawAddString_NotAllowed(t *testing.T) {

	table := newTestTable()
	table.CanAdd = false

	result, err := table.DrawAddString()

	require.NoError(t, err)
	// With adding disabled, the render falls back to view-only (no editable input row)
	assert.NotContains(t, result, `<input name="name"`)
}

func TestDrawAddString_Error(t *testing.T) {

	table := newTestTable()
	table.Schema = nil

	result, err := table.DrawAddString()

	require.Error(t, err)
	assert.Empty(t, result)
}

/******************************************
 * DrawEditString()
 ******************************************/

func TestDrawEditString(t *testing.T) {

	table := newTestTable()

	result, err := table.DrawEditString(1)

	require.NoError(t, err)
	assert.Contains(t, result, "<form")
	assert.Contains(t, result, `value="Sarah Connor"`) // row 1 loaded for editing
}

func TestDrawEditString_OutOfBounds(t *testing.T) {

	table := newTestTable()

	// An out-of-range edit index falls back to view-only mode (no <form>)
	result, err := table.DrawEditString(99)

	require.NoError(t, err)
	assert.Contains(t, result, `<div class="grid"`)
	assert.NotContains(t, result, "<form")
}

func TestDrawEditString_NegativeIndex(t *testing.T) {

	table := newTestTable()

	result, err := table.DrawEditString(-1)

	require.NoError(t, err)
	assert.NotContains(t, result, "<form") // negative index is out of bounds => view-only
}

func TestDrawEditString_Error(t *testing.T) {

	table := newTestTable()
	table.Schema = nil

	result, err := table.DrawEditString(0)

	require.Error(t, err)
	assert.Empty(t, result)
}

/******************************************
 * Field Rendering Errors
 *
 * A form that references an unregistered widget type causes field.View /
 * field.Edit to fail, which exercises the error branches inside
 * drawViewRow, drawAddRow, and drawEditRow.
 ******************************************/

// breakForm swaps in a form whose column uses an unregistered widget type,
// guaranteeing that field rendering fails.
func breakForm(table *Table) {
	table.Form = pointerTo(form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "this-widget-does-not-exist", Label: "Name", Path: "name"},
		},
	})
}

func TestDrawViewString_FieldError(t *testing.T) {

	table := newTestTable()
	breakForm(&table)

	result, err := table.DrawViewString()

	require.Error(t, err)
	assert.Empty(t, result)
}

func TestDrawAddString_FieldError(t *testing.T) {

	table := newTestTable()
	breakForm(&table)

	result, err := table.DrawAddString()

	require.Error(t, err)
	assert.Empty(t, result)
}

func TestDrawEditString_FieldError(t *testing.T) {

	table := newTestTable()
	breakForm(&table)

	result, err := table.DrawEditString(0)

	require.Error(t, err)
	assert.Empty(t, result)
}

/******************************************
 * Writer Errors
 ******************************************/

// errorWriter is an io.Writer that always fails, used to exercise the
// buffer.Write error branch in drawTable.
type errorWriter struct{}

func (errorWriter) Write(p []byte) (int, error) {
	return 0, assert.AnError
}

func TestDrawView_WriteError(t *testing.T) {

	table := newTestTable()

	err := table.DrawView(errorWriter{})

	require.Error(t, err)
}

/******************************************
 * Unreachable / Undocumented Paths
 *
 * A handful of error branches are not exercised by this suite because they
 * cannot be triggered through the public API with a well-formed schema:
 *
 *   - drawTable: the `tableSchema.Get(...)` per-row error never fires once
 *     getTableElement has already validated the array.
 *   - drawAddRow: the "Paranoid double-check" `if !widget.CanAdd` guard is
 *     unreachable because drawTable only calls it when CanAdd is true.
 *   - drawEditRow: the "Editing is not allowed. THIS SHOULD NEVER HAPPEN"
 *     guard is unreachable because drawTable only calls it when CanEdit is true.
 *   - DoEdit: the `Schema.Get(...)` / `Schema.Set(...)` failure paths require a
 *     schema/object mismatch that the table's own schema prevents.
 ******************************************/

// pointerTo returns a pointer to the given value. Used to build inline schemas.
func pointerTo[T any](value T) *T {
	return &value
}
