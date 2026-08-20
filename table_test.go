package table

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/benpate/form"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Test Setup / Shared Helpers
 ******************************************/

// TestMain registers all of the standard form widgets ("text", "number", etc)
// before running the test suite.  Without this, the form package cannot render
// any fields and every Draw* call fails with "Unrecognized form widget".
func TestMain(m *testing.M) {
	widget.UseAll()
	os.Exit(m.Run())
}

// testDatabase is a minimal object that satisfies the schema.PointerGetter
// interface so that the schema package can read/write the table data.
type testDatabase struct {
	Data sliceof.Object[mapof.Any]
}

// GetPointer implements the schema.PointerGetter interface.
func (d *testDatabase) GetPointer(name string) (any, bool) {
	if name == "data" {
		return &d.Data, true
	}
	return nil, false
}

// testSchema returns a schema with a "data" array of {name, age} objects.
func testSchema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"data": schema.Array{
					MinLength: 1,
					MaxLength: 6,
					Items: schema.Object{
						Properties: schema.ElementMap{
							"name": schema.String{},
							"age":  schema.Integer{},
						},
					},
				},
				"notArray": schema.String{},
			},
		},
	}
}

// testForm returns a UI form that displays the "name" and "age" columns.
func testForm() form.Element {
	return form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "text", Label: "Name", Path: "name"},
			{Type: "text", Label: "Age", Path: "age"},
		},
	}
}

// testData returns a database pre-populated with two rows.
func testData() *testDatabase {
	return &testDatabase{
		Data: sliceof.Object[mapof.Any]{
			mapof.Any{"name": "John Connor", "age": 20},
			mapof.Any{"name": "Sarah Connor", "age": 45},
		},
	}
}

// newTestTable assembles a fully configured Table widget (2 rows of data).
func newTestTable() Table {
	s := testSchema()
	f := testForm()
	return New(&s, &f, testData(), "data", testIconProvider{}, "http://localhost/table")
}

// testLookupProvider is a no-op implementation of form.LookupProvider.
type testLookupProvider struct{}

func (testLookupProvider) Group(_ string) form.LookupGroup { return nil }

/******************************************
 * Package Example
 ******************************************/

// ExampleTable builds a read-only table and prints part of the generated markup.
//
// Two details are easy to get wrong, and the Output below is what catches them:
// the row data has to be in a type rosetta can walk into (a plain
// []map[string]any is not), and every column's Type has to name a form widget
// that is actually registered.
func ExampleTable() {

	// Data schema defines the layout of the data.
	s := schema.New(schema.Array{
		MaxLength: 10,
		Items: schema.Object{
			Properties: schema.ElementMap{
				"name": schema.String{},
				"age":  schema.Integer{},
			},
		},
	})

	// UI schema defines which fields are displayed, and in which order. The "text"
	// widget takes its input type from the data schema, so the "age" column still
	// renders as <input type="number">.
	f := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "text", Label: "Name", Path: "name"},
			{Type: "text", Label: "Age", Path: "age"},
		},
	}

	// Define some data to render.
	data := sliceof.Object[mapof.Any]{
		{"name": "John Connor", "age": 20},
		{"name": "Sarah Connor", "age": 45},
	}

	// Create the new table and render it as HTML. The last argument before the URL
	// is your own IconProvider, which supplies the markup for the row controls.
	table := New(&s, &f, &data, "", testIconProvider{}, "/update-form")

	result, err := table.DrawViewString()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print a digestible slice of the markup rather than all ~1.6kb of it.
	fmt.Println(strings.Count(result, `<tr class="grid-row`), "data rows")
	fmt.Println(result[strings.Index(result, "<tr") : strings.Index(result, "</tr>")+len("</tr>")])

	// Output:
	// 2 data rows
	// <tr class="grid-header"><td class="grid-cell"><div>Name</div></td><td class="grid-cell"><div>Age</div></td><td class="grid-cell grid-controls"></td></tr>
}

/******************************************
 * New()
 ******************************************/

func TestNew(t *testing.T) {

	s := testSchema()
	f := testForm()
	data := testData()
	icons := testIconProvider{}

	table := New(&s, &f, data, "data", icons, "http://localhost/table")

	assert.Same(t, &s, table.Schema)
	assert.Same(t, &f, table.Form)
	assert.Equal(t, data, table.Object)
	assert.Equal(t, "data", table.Path)
	assert.Equal(t, "http://localhost/table", table.TargetURL)
	assert.Equal(t, icons, table.Icons)

	// New() grants all write permissions by default
	assert.True(t, table.CanAdd)
	assert.True(t, table.CanEdit)
	assert.True(t, table.CanDelete)

	// LookupProvider is not set by New()
	assert.Nil(t, table.LookupProvider)
}

/******************************************
 * Configuration Methods
 ******************************************/

// The builders return a modified copy and leave the original untouched, so each
// test asserts both the returned value and that the receiver is unchanged.

func TestAllowAdd(t *testing.T) {
	table := newTestTable()
	table.CanAdd = false

	result := table.AllowAdd()

	assert.True(t, result.CanAdd) // the returned copy allows adding
	assert.False(t, table.CanAdd) // the original is left unchanged
}

func TestAllowEdit(t *testing.T) {
	table := newTestTable()
	table.CanEdit = false

	result := table.AllowEdit()

	assert.True(t, result.CanEdit)
	assert.False(t, table.CanEdit)
}

func TestAllowDelete(t *testing.T) {
	table := newTestTable()
	table.CanDelete = false

	result := table.AllowDelete()

	assert.True(t, result.CanDelete)
	assert.False(t, table.CanDelete)
}

func TestAllowAll(t *testing.T) {
	table := newTestTable()
	table.CanAdd = false
	table.CanEdit = false
	table.CanDelete = false

	result := table.AllowAll()

	assert.True(t, result.CanAdd)
	assert.True(t, result.CanEdit)
	assert.True(t, result.CanDelete)

	// The original is left unchanged
	assert.False(t, table.CanAdd)
	assert.False(t, table.CanEdit)
	assert.False(t, table.CanDelete)
}

func TestAllowNone(t *testing.T) {
	table := newTestTable() // New() grants all permissions

	result := table.AllowNone()

	assert.False(t, result.CanAdd)
	assert.False(t, result.CanEdit)
	assert.False(t, result.CanDelete)

	// The original is left unchanged
	assert.True(t, table.CanAdd)
	assert.True(t, table.CanEdit)
	assert.True(t, table.CanDelete)
}

func TestUseLookupProvider(t *testing.T) {
	table := newTestTable()
	provider := testLookupProvider{}

	result := table.UseLookupProvider(provider)

	assert.Equal(t, provider, result.LookupProvider)
	assert.Nil(t, table.LookupProvider) // the original is left unchanged
}

// The builders use value receivers so they can be chained directly off New
// without the result escaping to the heap.
func TestNew_BuildersChainOffConstructor(t *testing.T) {
	s := testSchema()
	f := testForm()

	table := New(&s, &f, testData(), "data", testIconProvider{}, "http://localhost/table").
		AllowNone().
		UseLookupProvider(testLookupProvider{})

	assert.False(t, table.CanAdd)
	assert.False(t, table.CanEdit)
	assert.False(t, table.CanDelete)
	assert.NotNil(t, table.LookupProvider)
}

/******************************************
 * getURL()
 ******************************************/

func TestGetURL(t *testing.T) {

	table := newTestTable() // TargetURL == "http://localhost/table"

	// check is a closure-driven test that confirms a single getURL call.
	check := func(action string, row int, col int, expected string) {
		assert.Equal(t, expected, table.getURL(action, row, col), "action=%s row=%d col=%d", action, row, col)
	}

	check("add", 0, 0, "http://localhost/table?add=true")
	check("add", 5, 9, "http://localhost/table?add=true") // row/col ignored for "add"

	check("edit", 0, 0, "http://localhost/table?edit=0&focus=0")
	check("edit", 3, 2, "http://localhost/table?edit=3&focus=2")

	check("delete", 0, 0, "http://localhost/table?delete=0")
	check("delete", 7, 4, "http://localhost/table?delete=7") // col ignored for "delete"

	// Unrecognized actions return the bare TargetURL
	check("", 0, 0, "http://localhost/table")
	check("unknown", 1, 1, "http://localhost/table")
}

// When the TargetURL already carries a query string, getURL must merge its
// parameters in rather than appending a second "?".
func TestGetURL_TargetWithExistingQuery(t *testing.T) {

	table := newTestTable()
	table.TargetURL = "http://localhost/table?section=tasks"

	check := func(action string, row int, col int, expected string) {
		assert.Equal(t, expected, table.getURL(action, row, col), "action=%s row=%d col=%d", action, row, col)
	}

	// url.Values.Encode sorts keys alphabetically, so the existing "section" param is preserved
	check("add", 0, 0, "http://localhost/table?add=true&section=tasks")
	check("edit", 3, 2, "http://localhost/table?edit=3&focus=2&section=tasks")
	check("delete", 7, 0, "http://localhost/table?delete=7&section=tasks")

	// Unrecognized actions still return the bare TargetURL, untouched
	check("unknown", 0, 0, "http://localhost/table?section=tasks")
}

// A TargetURL that url.Parse rejects makes getURL fall back to returning the raw
// string. TargetURL is a plain string field, so this is reachable by simple
// misconfiguration -- not a theoretical branch.
func TestGetURL_UnparseableTarget(t *testing.T) {

	table := newTestTable()

	// Each of these fails url.Parse for a different reason: an unclosed IPv6
	// literal, a missing scheme, a raw control character, a space in the host,
	// and a bad percent-escape.
	for _, target := range []string{"http://[::1", "://x", "\x7f", "http://a b.com", "%zz"} {

		table.TargetURL = target

		// Every action falls back to the unmodified TargetURL, with no panic.
		assert.Equal(t, target, table.getURL("add", 0, 0), "target=%q", target)
		assert.Equal(t, target, table.getURL("edit", 1, 2), "target=%q", target)
		assert.Equal(t, target, table.getURL("delete", 3, 0), "target=%q", target)
	}
}

// An empty TargetURL parses successfully (into an empty URL), so the action's
// query params are still appended.
func TestGetURL_EmptyTarget(t *testing.T) {

	table := newTestTable()
	table.TargetURL = ""

	assert.Equal(t, "?add=true", table.getURL("add", 0, 0))
	assert.Equal(t, "", table.getURL("unknown", 0, 0))
}

// FuzzGetURL throws arbitrary target URLs and actions at getURL. It never panics,
// and an unrecognized action always returns the TargetURL completely untouched --
// the property callers rely on to detect "no action".
func FuzzGetURL(f *testing.F) {

	f.Add("http://localhost/table", "add", 0, 0)
	f.Add("http://localhost/table?a=b", "edit", 1, 2)
	f.Add("", "delete", -1, -1)
	f.Add("http://[::1", "add", 0, 0)
	f.Add("://x", "edit", 0, 0)
	f.Add("%zz", "", 0, 0)
	f.Add("http://x/\u00e9\u00e9", "unknown", 2147483647, -2147483648)

	f.Fuzz(func(t *testing.T, target string, action string, row int, col int) {

		table := newTestTable()
		table.TargetURL = target

		// Called for every action, not just the asserted ones -- the point is that
		// getURL survives an arbitrary TargetURL whatever the action. Moving this
		// into the `if` (as scopeguard suggests) would stop exercising the parse
		// path for "add", "edit", and "delete" entirely.
		result := table.getURL(action, row, col) // nolint:scopeguard

		// An action getURL does not recognize is echoed back verbatim.
		if action != "add" && action != "edit" && action != "delete" {
			require.Equal(t, target, result, "unrecognized action %q must not rewrite the URL", action)
		}
	})
}

/******************************************
 * Required-Field Contract
 *
 * Schema, Form, and Icons are documented as required, and New() takes all three.
 * Leaving one nil is a programming error, and the widget panics rather than
 * rendering something wrong. These tests pin that contract so the behavior is a
 * decision rather than an accident -- note that it is NOT uniform: a nil Schema
 * is reported as an error on the Draw path (getTableElement guards it) but
 * panics on the Do path.
 ******************************************/

func TestTable_NilSchemaPanicsInDoEdit(t *testing.T) {

	table := newTestTable()
	table.Schema = nil

	// Contrast with TestGetTableElement_NilSchema, where the Draw path returns a
	// clean error for exactly the same misconfiguration.
	assert.Panics(t, func() { _ = table.DoEdit(map[string]any{"name": "x"}, 0) })
}

func TestTable_NilFormPanicsInDraw(t *testing.T) {

	table := newTestTable()
	table.Form = nil

	assert.Panics(t, func() { _, _ = table.DrawViewString() })
}

func TestTable_NilIconsPanicsInDraw(t *testing.T) {

	table := newTestTable()
	table.Icons = nil

	// The icons are only reached once rows are being rendered.
	assert.Panics(t, func() { _, _ = table.DrawViewString() })
}

/******************************************
 * getTableElement()
 ******************************************/

func TestGetTableElement(t *testing.T) {

	table := newTestTable()

	element, err := table.getTableElement()

	require.NoError(t, err)
	assert.Equal(t, 6, element.MaxLength)
	assert.Equal(t, 1, element.MinLength)
}

func TestGetTableElement_NilSchema(t *testing.T) {

	table := newTestTable()
	table.Schema = nil

	element, err := table.getTableElement()

	require.Error(t, err)
	assert.Equal(t, schema.Array{}, element)
}

func TestGetTableElement_PathNotFound(t *testing.T) {

	table := newTestTable()
	table.Path = "missing"

	element, err := table.getTableElement()

	require.Error(t, err)
	assert.Equal(t, schema.Array{}, element)
}

func TestGetTableElement_NotAnArray(t *testing.T) {

	table := newTestTable()
	table.Path = "notArray" // points to a schema.String, not a schema.Array

	element, err := table.getTableElement()

	require.Error(t, err)
	assert.Equal(t, schema.Array{}, element)
}
