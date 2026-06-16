package table

import (
	"fmt"
	"os"
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
 * Original Example (kept for documentation)
 ******************************************/

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

	// UI schema defines which field are displayed, and in which order
	f := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "text", Label: "Name", Path: "name"},
			{Type: "number", Label: "Age", Path: "age"},
		},
	}

	// Define some data to render
	data := []map[string]any{
		{"name": "John Connor", "age": 20},
		{"name": "Sarah Connor", "age": 45},
	}

	// Create the new table and render it in HTML
	table := New(&s, &f, &data, "", testIconProvider{}, "http://localhost/update-form")
	fmt.Println(table.DrawViewString())
}

func TestTable(_ *testing.T) {
	ExampleTable()
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
