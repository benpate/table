package table

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/benpate/form"
	"github.com/benpate/html"
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

// autofocusedInput returns the rendered <input ...> tag that carries the
// "autofocus" attribute, or "" if no input is autofocused.
func autofocusedInput(html string) string {
	for _, chunk := range strings.Split(html, "<input") {
		if strings.Contains(chunk, "autofocus") {
			return chunk
		}
	}
	return ""
}

func TestDraw_FocusParam(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?edit=0&focus=1"), &buffer)

	require.NoError(t, err)
	// The focus query param autofocuses the requested column (column 1 == "age")
	assert.Contains(t, autofocusedInput(buffer.String()), `name="age"`)
}

// A focus column beyond the last column is untrusted input: it must not panic,
// and clamps back to the first column.
func TestDraw_FocusTooLarge(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?edit=0&focus=999"), &buffer)

	require.NoError(t, err)
	assert.Contains(t, autofocusedInput(buffer.String()), `name="name"`) // clamped to column 0
}

// A negative focus column must not panic.
func TestDraw_FocusNegative(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?add=true&focus=-1"), &buffer)

	require.NoError(t, err)
	// Adding always focuses the first column, and the bad focus value is harmless
	assert.Contains(t, autofocusedInput(buffer.String()), `name="name"`)
}

// An "edit" value too large to fit in an int falls through to a normal (view-only)
// render rather than panicking on the failed parse.
func TestDraw_EditOverflows(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?edit=999999999999999999999"), &buffer)

	require.NoError(t, err)
	result := buffer.String()
	assert.Contains(t, result, `<div class="grid"`) // view mode, not an edit <form>
	assert.NotContains(t, result, "<form")
}

// An "edit" index past the end of the table is out of bounds: drawTable falls back
// to view-only mode rather than rendering a (nonexistent) editable row.
func TestDraw_EditOutOfBounds(t *testing.T) {

	table := newTestTable() // 2 rows
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?edit=99"), &buffer)

	require.NoError(t, err)
	assert.NotContains(t, buffer.String(), "<form") // no editable row
}

// "add" takes precedence over "edit" when both are present.
func TestDraw_AddBeatsEdit(t *testing.T) {

	table := newTestTable()
	var buffer bytes.Buffer

	err := table.Draw(mustURL(t, "http://x?add=true&edit=0"), &buffer)

	require.NoError(t, err)
	// Adding renders an editable <form> with a blank first column (the new row),
	// not an edit of the existing row 0.
	result := buffer.String()
	assert.Contains(t, result, "<form")
}

// FuzzDraw confirms that Draw() never panics, regardless of the "add", "edit", and
// "focus" query-param values it has to parse and clamp.  It mirrors FuzzDo for the
// render side of the widget.
func FuzzDraw(f *testing.F) {

	f.Add("true", "0", "0")
	f.Add("", "0", "1")
	f.Add("", "abc", "")
	f.Add("", "999999999999999999999", "-1")
	f.Add("nope", "-1", "999")
	f.Add("", "", "")

	f.Fuzz(func(_ *testing.T, add string, edit string, focus string) {

		table := newTestTable()

		params := &url.URL{RawQuery: url.Values{
			"add":   {add},
			"edit":  {edit},
			"focus": {focus},
		}.Encode()}

		// We don't care whether Draw succeeds or fails, only that it does not panic.
		var buffer bytes.Buffer
		_ = table.Draw(params, &buffer)
	})
}

/******************************************
 * drawTable() - Header "column-width" Option
 *
 * A column's "column-width" option (form.Element.Options) becomes the CSS width of
 * its header cell, via mapof.Any.GetString.  These tests pin how each value type
 * renders, including the accepted lossy two-decimal formatting of a float width.
 ******************************************/

// columnWidthTable builds a Table whose single column carries the given
// "column-width" option (or no option at all when width is nil), then returns the
// rendered view-mode HTML.
func columnWidthTable(t *testing.T, width any) string {
	t.Helper()

	s := testSchema()

	options := mapof.Any{}
	if width != nil {
		options["column-width"] = width
	}

	f := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "text", Label: "Name", Path: "name", Options: options},
		},
	}

	table := New(&s, &f, testData(), "data", testIconProvider{}, "http://x")

	result, err := table.DrawViewString()
	require.NoError(t, err)
	return result
}

// A string width (e.g. a percentage) is rendered verbatim onto the header cell.
func TestDrawColumnWidth_String(t *testing.T) {
	assert.Contains(t, columnWidthTable(t, "50%"), `style="width; 50%"`)
}

// An integer width is rendered as a plain integer (no decimals).
func TestDrawColumnWidth_Integer(t *testing.T) {
	assert.Contains(t, columnWidthTable(t, 200), `style="width; 200"`)
}

// A float width is rendered with two decimal places.  This is lossy (rosetta's
// convert.StringOk formats floats to two decimals), which is acceptable for a
// sub-pixel column width.  Pinned so a future rosetta change is caught.
func TestDrawColumnWidth_FloatIsTwoDecimals(t *testing.T) {
	assert.Contains(t, columnWidthTable(t, 33.333), `style="width; 33.33"`)
	assert.Contains(t, columnWidthTable(t, float64(150)), `style="width; 150.00"`)
}

// When the column carries no "column-width" option, the header cell gets no width
// style at all.
func TestDrawColumnWidth_Absent(t *testing.T) {
	result := columnWidthTable(t, nil)
	assert.Contains(t, result, `<td class="grid-cell"><div>Name</div>`) // header cell, no style attr
}

// An empty-string width is treated as "no width": GetString returns "", which the
// width != "" guard skips, so no width style is emitted.
func TestDrawColumnWidth_EmptyString(t *testing.T) {
	result := columnWidthTable(t, "")
	assert.Contains(t, result, `<td class="grid-cell"><div>Name</div>`) // no style attr
}

// sharedForm returns a schema + form whose columns carry non-nil Options maps,
// so that any accidental write to the shared form is observable.
func sharedForm() (schema.Schema, form.Element) {
	return testSchema(), form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "text", Label: "Name", Path: "name", Options: mapof.Any{}},
			{Type: "text", Label: "Age", Path: "age", Options: mapof.Any{}},
		},
	}
}

// Rendering must never write the transient "focus" flag back into the shared
// Form definition.  Adding focuses the first column; it must do so on a copy.
func TestDrawAdd_DoesNotMutateSharedForm(t *testing.T) {

	s, f := sharedForm()
	table := New(&s, &f, testData(), "data", testIconProvider{}, "http://x")

	_, err := table.DrawAddString()

	require.NoError(t, err)
	_, ok := f.Children[0].Options["focus"]
	assert.False(t, ok, "add render must not leave 'focus' on the shared form")
}

// Editing with a non-default focus column must focus that column on a copy,
// never writing "focus" back into any column of the shared Form.
func TestDrawEdit_DoesNotMutateSharedForm(t *testing.T) {

	s, f := sharedForm()
	table := New(&s, &f, testData(), "data", testIconProvider{}, "http://x")

	var buffer bytes.Buffer
	err := table.Draw(mustURL(t, "http://x?edit=0&focus=1"), &buffer)

	require.NoError(t, err)
	for index, child := range f.Children {
		_, ok := child.Options["focus"]
		assert.False(t, ok, "edit render must not leave 'focus' on shared form column %d", index)
	}
}

// Rendering the same shared *form.Element from many goroutines must be race-free.
// Each goroutine owns its own Table but they all share one schema and form, and
// each focuses a different column via the public Draw API. Run with -race.
func TestDraw_SharedFormIsRaceFree(t *testing.T) {

	s, f := sharedForm()

	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		params := mustURL(t, "http://x?edit=0&focus="+strconv.Itoa(i%len(f.Children)))
		wg.Add(1)
		go func(p *url.URL) {
			defer wg.Done()
			var buffer bytes.Buffer
			table := New(&s, &f, testData(), "data", testIconProvider{}, "http://x")
			_ = table.Draw(p, &buffer)
		}(params)
	}
	wg.Wait()
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

	table := newTestTable().AllowNone()

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
	assert.NotContains(t, result, "Add a Row")                                                // the add control is omitted for this render...
	assert.True(t, table.CanAdd, "render must not mutate the widget's configured permission") // ...without mutating the widget
}

func TestDrawViewString_MinLengthDisablesDelete(t *testing.T) {

	// A table that is at (or below) MinLength cannot delete rows
	table := newTestTable()
	db := table.Object.(*testDatabase)
	db.Data = sliceof.Object[mapof.Any]{mapof.Any{"name": "Last One", "age": 99}} // single row, MinLength == 1

	result, err := table.DrawViewString()

	require.NoError(t, err)
	assert.NotContains(t, result, "hx-confirm")                                                  // the delete control is omitted for this render...
	assert.True(t, table.CanDelete, "render must not mutate the widget's configured permission") // ...without mutating the widget
}

func TestDrawViewString_Error(t *testing.T) {

	table := newTestTable()
	table.Schema = nil // getTableElement will fail

	result, err := table.DrawViewString()

	require.Error(t, err)
	assert.Empty(t, result)
}

// A misconfigured Object (not a schema.PointerGetter) makes Schema.Get fail.
// That error must be reported, not silently rendered as an empty table.
func TestDrawViewString_BadObject(t *testing.T) {

	table := newTestTable()
	table.Object = "not a pointer-getter" // Schema.Get will fail to read the data

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

// TestDrawEditString_BalancedRowTags guards the row markup emitted in EDIT mode.
// drawEditRow once closed only its final <td>, leaving the <tr> open so the next
// row nested inside it -- so every row is counted here, not just inspected for
// content.
func TestDrawEditString_BalancedRowTags(t *testing.T) {

	table := newTestTable()

	result, err := table.DrawEditString(0)

	require.NoError(t, err)

	// The test table has 2 rows, so the output is a header row plus 2 data rows.
	// Every opening tag must have exactly one match.
	assert.Equal(t, 3, strings.Count(result, "<tr"), "expected 3 opening <tr> tags")
	assert.Equal(t, 3, strings.Count(result, "</tr>"), "every <tr> must be closed")
	assert.Equal(t, strings.Count(result, "<td"), strings.Count(result, "</td>"), "every <td> must be closed")

	// A row may never open inside another row: the edit row has to close before
	// the following view row opens.
	editRow := result[strings.Index(result, "grid-editable"):]
	assert.Less(t, strings.Index(editRow, "</tr>"), strings.Index(editRow, "<tr"), "the edit row must close before the next row opens")
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

func (errorWriter) Write(_ []byte) (int, error) {
	return 0, assert.AnError
}

func TestDrawView_WriteError(t *testing.T) {

	table := newTestTable()

	err := table.DrawView(errorWriter{})

	require.Error(t, err)
}

/******************************************
 * focusField()
 ******************************************/

// focusField must copy the column's existing options across, not just set "focus"
// on a fresh map -- otherwise focusing a column silently drops its configuration
// (column-width, and anything else the caller set).
func TestFocusField_PreservesExistingOptions(t *testing.T) {

	field := form.Element{
		Type:    "text",
		Label:   "Name",
		Path:    "name",
		Options: mapof.Any{"column-width": "50%", "custom": 42},
	}

	result := focusField(field)

	// The focus flag is added...
	assert.Equal(t, true, result.Options["focus"])

	// ...and every pre-existing option survives.
	assert.Equal(t, "50%", result.Options["column-width"])
	assert.Equal(t, 42, result.Options["custom"])

	// The original field's map is untouched (no "focus" leaked back into it).
	_, ok := field.Options["focus"]
	assert.False(t, ok)
	assert.Len(t, field.Options, 2)
}

// A column with no Options at all still gets a usable map back.
func TestFocusField_NilOptions(t *testing.T) {

	result := focusField(form.Element{Type: "text", Path: "name"})

	assert.Equal(t, true, result.Options["focus"])
	assert.Len(t, result.Options, 1)
}

// Focusing a column whose options are shared must not write through to the
// original map, since Form definitions are shared across concurrent renders.
func TestFocusField_DoesNotAliasSharedMap(t *testing.T) {

	shared := mapof.Any{"column-width": "10%"}
	field := form.Element{Type: "text", Path: "name", Options: shared}

	result := focusField(field)
	result.Options["column-width"] = "MUTATED"

	assert.Equal(t, "10%", shared["column-width"], "focusField must deep-copy the options map")
}

/******************************************
 * Row Guards (white-box)
 *
 * drawTable only calls these helpers when the matching permission is already
 * true, so the guards cannot fire through the public API. They are called
 * directly here to pin what they do if that ever changes.
 ******************************************/

// rowSchemaFor returns a standalone row schema for calling the row helpers directly.
func rowSchemaFor() schema.Schema {
	return schema.New(schema.Object{
		Properties: schema.ElementMap{"name": schema.String{}, "age": schema.Integer{}},
	})
}

// drawAddRow refuses silently: it writes nothing and reports no error.
func TestDrawAddRow_GuardWhenNotAllowed(t *testing.T) {

	table := newTestTable()
	rowSchema := rowSchemaFor()
	builder := html.New()

	err := table.drawAddRow(&rowSchema, false, builder)

	require.NoError(t, err)
	assert.Empty(t, builder.String(), "the guard must not emit any markup")
}

// drawEditRow refuses loudly, because silently rendering a non-editable row as
// editable would be a permissions failure rather than a cosmetic one.
func TestDrawEditRow_GuardWhenNotAllowed(t *testing.T) {

	table := newTestTable()
	rowSchema := rowSchemaFor()
	builder := html.New()

	err := table.drawEditRow(&rowSchema, nil, false, 0, builder)

	require.Error(t, err)
	assert.Empty(t, builder.String(), "the guard must not emit any markup")
}

/******************************************
 * Empty and Degenerate Tables
 ******************************************/

// A table with no rows renders its header and (when permitted) the add control,
// rather than failing or emitting a malformed table.
func TestDrawView_EmptyTable(t *testing.T) {

	table := newTestTable()
	table.Object.(*testDatabase).Data = sliceof.NewObject[mapof.Any]()

	result, err := table.DrawViewString()

	require.NoError(t, err)
	assert.Contains(t, result, "<div>Name</div>")      // header still renders
	assert.Equal(t, 1, strings.Count(result, "<tr"))   // header row only
	assert.Equal(t, 1, strings.Count(result, "</tr>")) // ...and it is closed
	assert.NotContains(t, result, `class="grid-row`)   // no data rows
	assert.Contains(t, result, "plus Add a Row")       // adding is still offered
	assert.NotContains(t, result, "hx-confirm")        // nothing to delete
}

// An empty table in ADD mode renders exactly one editable row.
func TestDrawAdd_EmptyTable(t *testing.T) {

	table := newTestTable()
	table.Object.(*testDatabase).Data = sliceof.NewObject[mapof.Any]()

	result, err := table.DrawAddString()

	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(result, "<tr"))   // header + the new row
	assert.Equal(t, 2, strings.Count(result, "</tr>")) // both closed
	assert.Contains(t, result, `<input name="name"`)
}

// With no rows to draw, a broken Form surfaces from drawAddRow rather than from
// drawViewRow -- the only way to reach drawAddRow's field-rendering error.
func TestDrawAdd_EmptyTableFieldError(t *testing.T) {

	table := newTestTable()
	table.Object.(*testDatabase).Data = sliceof.NewObject[mapof.Any]()
	breakForm(&table)

	result, err := table.DrawAddString()

	require.Error(t, err)
	assert.Empty(t, result)
}

// A Form with no columns is degenerate but must not panic: the per-column width
// divides by the column count, and the focus clamp indexes the column list.
func TestDrawView_ZeroColumnForm(t *testing.T) {

	table := newTestTable()
	table.Form = pointerTo(form.Element{Type: "layout-vertical", Children: []form.Element{}})

	result, err := table.DrawViewString()

	require.NoError(t, err)
	assert.Contains(t, result, `<table class="grid">`)
	assert.Equal(t, strings.Count(result, "<tr"), strings.Count(result, "</tr>"))
}

// The same degenerate form, reached through the public Draw router with a focus
// param, which is where the clamp against len(Children) actually runs.
func TestDraw_ZeroColumnFormWithFocus(t *testing.T) {

	table := newTestTable()
	table.Form = pointerTo(form.Element{Type: "layout-vertical", Children: []form.Element{}})

	var buffer bytes.Buffer
	err := table.Draw(mustURL(t, "http://x?edit=0&focus=3"), &buffer)

	require.NoError(t, err)
}

/******************************************
 * Output Escaping
 *
 * Row values come from stored data that a user typed, so they reach the renderer
 * untrusted. The escaping is done by the form/html packages rather than here, so
 * these tests pin the guarantee at this package's boundary -- a change in either
 * dependency that reopened an injection would otherwise pass silently.
 ******************************************/

// hostileTable returns a table whose single row carries the given "name" value.
func hostileTable(value string) Table {
	table := newTestTable()
	table.Object.(*testDatabase).Data = sliceof.Object[mapof.Any]{
		mapof.Any{"name": value, "age": 1},
	}
	return table
}

func TestDrawView_EscapesHostileValue(t *testing.T) {

	result, err := hostileTable(`<script>alert(1)</script>`).DrawViewString()

	require.NoError(t, err)
	assert.NotContains(t, result, "<script>")
	assert.Contains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
}

// In edit mode the value lands inside an attribute, where a bare quote would
// break out of value="..." -- a different escaping context from the view cell.
func TestDrawEdit_EscapesHostileValue(t *testing.T) {

	result, err := hostileTable(`" onfocus="alert(1)`).DrawEditString(0)

	require.NoError(t, err)
	assert.NotContains(t, result, `" onfocus="alert(1)"`)
	assert.Contains(t, result, "&#34;")
}

func TestDrawEdit_EscapesHostileScriptValue(t *testing.T) {

	result, err := hostileTable(`</textarea><script>alert(1)</script>`).DrawEditString(0)

	require.NoError(t, err)
	assert.NotContains(t, result, "<script>")
}

// Column labels come from the Form definition rather than from a user, but they
// are rendered into the header cell and escaped all the same.
func TestDrawView_EscapesColumnLabel(t *testing.T) {

	table := newTestTable()
	table.Form = pointerTo(form.Element{
		Type:     "layout-vertical",
		Children: []form.Element{{Type: "text", Label: `<img src=x onerror=alert(1)>`, Path: "name"}},
	})

	result, err := table.DrawViewString()

	require.NoError(t, err)
	assert.NotContains(t, result, "<img src=x")
	assert.Contains(t, result, "&lt;img")
}

/******************************************
 * Unusual Row Values
 ******************************************/

// Invalid UTF-8 in stored data must render without panicking or truncating the
// document. Go writes the raw bytes through, so the assertion is only that the
// surrounding markup is intact.
func TestDrawView_InvalidUTF8Value(t *testing.T) {

	result, err := hostileTable("bad\xff\xfe\x00utf8").DrawViewString()

	require.NoError(t, err)
	assert.Contains(t, result, "<div>Name</div>")
	assert.Equal(t, strings.Count(result, "<tr"), strings.Count(result, "</tr>"))
}

// A very long value is rendered in full (the widget imposes no truncation of its
// own), and does not corrupt the surrounding markup.
func TestDrawView_VeryLongValue(t *testing.T) {

	long := strings.Repeat("A", 100_000)

	result, err := hostileTable(long).DrawViewString()

	require.NoError(t, err)
	assert.Contains(t, result, long)
	assert.Equal(t, strings.Count(result, "<tr"), strings.Count(result, "</tr>"))
}

// A table filled to its schema MaxLength renders every row and drops the add
// control, with the row markup still balanced.
func TestDrawView_AtMaxLength(t *testing.T) {

	table := newTestTable() // schema MaxLength == 6
	rows := sliceof.NewObject[mapof.Any]()
	for i := 0; i < 6; i++ {
		rows = append(rows, mapof.Any{"name": "row-" + strconv.Itoa(i), "age": i})
	}
	table.Object.(*testDatabase).Data = rows

	result, err := table.DrawViewString()

	require.NoError(t, err)
	assert.Equal(t, 7, strings.Count(result, "<tr"))   // header + 6 rows
	assert.Equal(t, 7, strings.Count(result, "</tr>")) // all closed
	assert.NotContains(t, result, "Add a Row")         // at capacity
}

// Row data must be held in a type rosetta can index into. A plain
// []map[string]any is not one: the top-level Schema.Get succeeds (it returns the
// slice) and the row count is right, but reading row "0" back out fails. This is
// the failure that a table configured from ordinary Go maps hits, so it is
// reported rather than rendered as an empty table.
func TestDrawView_UnindexableRowData(t *testing.T) {

	rowSchema := schema.Object{Properties: schema.ElementMap{"name": schema.String{}, "age": schema.Integer{}}}
	s := schema.New(schema.Array{MaxLength: 10, Items: rowSchema})
	f := testForm()

	data := []map[string]any{{"name": "John Connor", "age": 20}}
	table := New(&s, &f, &data, "", testIconProvider{}, "http://x")

	result, err := table.DrawViewString()

	require.Error(t, err)
	assert.Empty(t, result)

	// The same schema and data work when the slice is one rosetta understands.
	ok := sliceof.Object[mapof.Any]{mapof.Any{"name": "John Connor", "age": 20}}
	table.Object = &ok

	result, err = table.DrawViewString()

	require.NoError(t, err)
	assert.Contains(t, result, "John Connor")
}

/******************************************
 * Shared Helpers
 ******************************************/

// pointerTo returns a pointer to the given value. Used to build inline schemas.
func pointerTo[T any](value T) *T {
	return &value
}
