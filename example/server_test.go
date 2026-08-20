package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/benpate/form/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Test Setup
 ******************************************/

// TestMain registers the standard form widgets exactly as main() does, and
// resets the in-memory database before each run.
func TestMain(m *testing.M) {
	widget.UseAll()
	os.Exit(m.Run())
}

// reset restores the demo database to its starting state, since the handlers
// mutate a package-level global.
func reset(t *testing.T) {
	t.Helper()
	database = getDefaultTableData()
}

/******************************************
 * The Demo Actually Renders
 *
 * The whole point of the example is that someone can run it and see a table.
 * That silently stopped being true once -- main() rendered an error page
 * because it never registered the form widgets -- so these tests assert the
 * demo produces real markup, not merely that it compiles.
 ******************************************/

func TestGetTable_RendersView(t *testing.T) {
	reset(t)

	result, err := getTable().DrawViewString()

	require.NoError(t, err)
	assert.Contains(t, result, "<div>Task Name</div>")
	assert.Contains(t, result, "Grocery Store")
	assert.Contains(t, result, "Hardware Store")

	// Row markup is balanced: a header row plus the two seeded rows.
	assert.Equal(t, 3, strings.Count(result, "<tr"))
	assert.Equal(t, 3, strings.Count(result, "</tr>"))
}

func TestGetTable_RendersAdd(t *testing.T) {
	reset(t)

	result, err := getTable().DrawAddString()

	require.NoError(t, err)
	assert.Contains(t, result, "<form")
	assert.Contains(t, result, `<input name="label"`)
	assert.Equal(t, strings.Count(result, "<tr"), strings.Count(result, "</tr>"))
}

func TestGetTable_RendersEdit(t *testing.T) {
	reset(t)

	result, err := getTable().DrawEditString(0)

	require.NoError(t, err)
	assert.Contains(t, result, "<form")
	assert.Contains(t, result, `value="Grocery Store"`)
	assert.Equal(t, strings.Count(result, "<tr"), strings.Count(result, "</tr>"))
}

// Every column in the demo Form names a widget that UseAll actually registers.
// A typo here is invisible until the page is loaded.
func TestGetTableForm_AllWidgetTypesAreRegistered(t *testing.T) {
	reset(t)

	// Rendering exercises every column; an unregistered Type fails the render.
	_, err := getTable().DrawAddString()

	require.NoError(t, err, "every column Type must name a registered form widget")
}

/******************************************
 * HTTP Handlers
 ******************************************/

func TestHandleTable_Get(t *testing.T) {
	reset(t)

	recorder := httptest.NewRecorder()
	handleTable()(recorder, httptest.NewRequest(http.MethodGet, "/table", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Grocery Store")
}

func TestHandleTable_GetAddMode(t *testing.T) {
	reset(t)

	recorder := httptest.NewRecorder()
	handleTable()(recorder, httptest.NewRequest(http.MethodGet, "/table?add=true", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "<form")
}

func TestHandleTable_PostEdit(t *testing.T) {
	reset(t)

	body := url.Values{
		"label":       {"Updated Errand"},
		"description": {"Changed."},
		"status":      {"Complete"},
		"assignedTo":  {"Carl"},
	}
	request := httptest.NewRequest(http.MethodPost, "/table?edit=0", strings.NewReader(body.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	handleTable()(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "Updated Errand", database.Data[0]["label"])
	assert.Contains(t, recorder.Body.String(), "Updated Errand")
}

func TestHandleTable_PostDelete(t *testing.T) {
	reset(t)

	request := httptest.NewRequest(http.MethodPost, "/table?delete=1", strings.NewReader(""))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	handleTable()(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, len(database.Data))
	assert.Equal(t, "Grocery Store", database.Data[0]["label"])
}

// A POST that fails to apply reports the error rather than rendering a table.
func TestHandleTable_PostError(t *testing.T) {
	reset(t)

	// The demo schema has MinLength 1, so deleting down to zero rows fails.
	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/table?delete=0", strings.NewReader(""))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		handleTable()(httptest.NewRecorder(), request)
	}

	request := httptest.NewRequest(http.MethodPost, "/table?delete=0", strings.NewReader(""))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handleTable()(recorder, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
}

// A POST whose body cannot be parsed as a form fails in bind(), before any table
// work happens, and is reported as an error response.
func TestHandleTable_PostMalformedBody(t *testing.T) {
	reset(t)

	request := httptest.NewRequest(http.MethodPost, "/table?edit=0", strings.NewReader("%zz=broken"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	handleTable()(recorder, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "Grocery Store", database.Data[0]["label"], "a malformed body must not modify the database")
}

/******************************************
 * Uncovered Paths
 *
 * Two branches in this file are deliberately left uncovered:
 *
 *   - main() binds a listening socket and blocks, so exercising it would mean
 *     restructuring the demo around an injectable http.Server. Not worth it for
 *     example code; the handlers it wires up are all tested individually above.
 *
 *   - handleTable's Draw error branch needs the render itself to fail, which for
 *     this demo means an unregistered form widget. Widget registration is global
 *     process state (form.Use), so deregistering one to force the error would
 *     make every other test in this package order-dependent.
 ******************************************/

// getFile serves a file from the working directory, and reports a missing one
// as an error rather than an empty 200.
func TestGetFile(t *testing.T) {

	recorder := httptest.NewRecorder()
	getFile("index.html")(recorder, httptest.NewRequest(http.MethodGet, "/index.html", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEmpty(t, recorder.Body.String())
}

func TestGetFile_Missing(t *testing.T) {

	recorder := httptest.NewRecorder()
	getFile("does-not-exist.html")(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

/******************************************
 * Helpers
 ******************************************/

func TestBind(t *testing.T) {

	body := url.Values{"label": {"Groceries"}, "status": {"New"}}
	request := httptest.NewRequest(http.MethodPost, "/table", strings.NewReader(body.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := bind(request)

	require.NoError(t, err)
	assert.Equal(t, "Groceries", result["label"])
	assert.Equal(t, "New", result["status"])
}

// bind keeps only the FIRST value when a key is repeated.
func TestBind_RepeatedKeyKeepsFirst(t *testing.T) {

	request := httptest.NewRequest(http.MethodPost, "/table", strings.NewReader("label=first&label=second"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := bind(request)

	require.NoError(t, err)
	assert.Equal(t, "first", result["label"])
}

func TestBind_MalformedBody(t *testing.T) {

	request := httptest.NewRequest(http.MethodPost, "/table", strings.NewReader("%zz=broken"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := bind(request)

	require.Error(t, err)
	assert.Empty(t, result)
}

func TestIconProvider_Get(t *testing.T) {

	provider := IconProvider{}

	// Known icons return Bootstrap Icons markup...
	for _, name := range []string{"plus", "edit", "delete", "save", "cancel"} {
		assert.Contains(t, provider.Get(name), "<i class=\"bi bi-", "icon=%s", name)
	}

	// ...and an unknown icon falls back to its own name.
	assert.Equal(t, "no-such-icon", provider.Get("no-such-icon"))
}

func TestIconProvider_Write(t *testing.T) {

	var builder strings.Builder
	IconProvider{}.Write("plus", &builder)

	assert.Equal(t, IconProvider{}.Get("plus"), builder.String())
}

func TestDatabase_GetPointer(t *testing.T) {

	db := getDefaultTableData()

	pointer, ok := db.GetPointer("data")
	require.True(t, ok)
	assert.Same(t, &db.Data, pointer)

	// Any other name is not addressable.
	_, ok = db.GetPointer("nope")
	assert.False(t, ok)
}

func TestGetDefaultTableData(t *testing.T) {

	db := getDefaultTableData()

	require.Equal(t, 2, len(db.Data))
	assert.Equal(t, "Grocery Store", db.Data[0]["label"])
	assert.Equal(t, "Alice", db.Data[1]["assignedTo"])
}

// The demo's schema and form must agree: every Form column must name a property
// that the data schema actually declares.
func TestGetTableSchema_MatchesForm(t *testing.T) {

	tableSchema := getTableSchema()
	element, ok := tableSchema.GetElement("data")
	require.True(t, ok)

	for _, column := range getTableForm().Children {
		_, found := tableSchema.GetElement("data.0." + column.Path)
		assert.True(t, found, "form column %q has no matching schema property", column.Path)
	}

	assert.NotNil(t, element)
}

func TestWriteError(t *testing.T) {

	recorder := httptest.NewRecorder()
	writeError(recorder, assertAnError{})

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "boom")
}

// assertAnError is a minimal error for exercising writeError.
type assertAnError struct{}

func (assertAnError) Error() string { return "boom" }
