package table

import (
	"net/url"
	"testing"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Fuzz Testing
 ******************************************/

// FuzzDo confirms that Do() never panics, regardless of the "edit" and
// "delete" query-param values it has to parse into row indices.
func FuzzDo(f *testing.F) {

	f.Add("0", "")
	f.Add("", "0")
	f.Add("abc", "")
	f.Add("999999999999999999999", "")
	f.Add("-1", "-1")
	f.Add("", "xyz")

	f.Fuzz(func(_ *testing.T, edit string, deleteVal string) {

		table := newTestTable()

		params := &url.URL{RawQuery: url.Values{"edit": {edit}, "delete": {deleteVal}}.Encode()}

		// We don't care whether Do succeeds or fails, only that it does not panic.
		_ = table.Do(params, map[string]any{"name": "fuzz", "age": 1})
	})
}

/******************************************
 * Do() - Query Param Router
 ******************************************/

// mustURL parses a raw URL or fails the test.
func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	result, err := url.Parse(raw)
	require.NoError(t, err)
	return result
}

func TestDo_Edit(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	err := table.Do(mustURL(t, "http://x?edit=0"), map[string]any{"name": "Kyle Reese", "age": 30})

	require.NoError(t, err)
	assert.Equal(t, "Kyle Reese", db.Data[0]["name"])
}

func TestDo_Delete(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	err := table.Do(mustURL(t, "http://x?delete=0"), nil)

	require.NoError(t, err)
	require.Equal(t, 1, len(db.Data))
	assert.Equal(t, "Sarah Connor", db.Data[0]["name"])
}

func TestDo_NoAction(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	err := table.Do(mustURL(t, "http://x"), map[string]any{"name": "ignored"})

	require.NoError(t, err)
	// Data is untouched
	assert.Equal(t, 2, len(db.Data))
	assert.Equal(t, "John Connor", db.Data[0]["name"])
}

func TestDo_EditNotNumeric(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	// A non-numeric "edit" value is silently ignored (Atoi fails, no error returned)
	err := table.Do(mustURL(t, "http://x?edit=abc"), map[string]any{"name": "ignored"})

	require.NoError(t, err)
	assert.Equal(t, "John Connor", db.Data[0]["name"])
}

func TestDo_DeleteNotNumeric(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	err := table.Do(mustURL(t, "http://x?delete=abc"), nil)

	require.NoError(t, err)
	assert.Equal(t, 2, len(db.Data))
}

func TestDo_EditError(t *testing.T) {

	table := newTestTable()

	// edit index is far beyond the end of the table => DoEdit fails => Do wraps the error
	err := table.Do(mustURL(t, "http://x?edit=99"), map[string]any{"name": "boom"})

	require.Error(t, err)
}

func TestDo_DeleteError(t *testing.T) {

	table := newTestTable()
	table.CanDelete = false

	// Deleting is not allowed => DoDelete fails => Do wraps the error
	err := table.Do(mustURL(t, "http://x?delete=0"), nil)

	require.Error(t, err)
}

/******************************************
 * DoEdit()
 ******************************************/

func TestDoEdit_ExistingRow(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	err := table.DoEdit(map[string]any{"name": "Miles Dyson", "age": 40}, 1)

	require.NoError(t, err)
	require.Equal(t, 2, len(db.Data)) // length unchanged
	assert.Equal(t, "Miles Dyson", db.Data[1]["name"])
}

func TestDoEdit_AppendNewRow(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	// editIndex == length means "add a new row at the end"
	err := table.DoEdit(map[string]any{"name": "T-800", "age": 0}, 2)

	require.NoError(t, err)
	require.Equal(t, 3, len(db.Data))
	assert.Equal(t, "T-800", db.Data[2]["name"])
}

func TestDoEdit_AppendNotAllowed(t *testing.T) {

	table := newTestTable()
	table.CanAdd = false

	err := table.DoEdit(map[string]any{"name": "T-800"}, 2) // index == length

	require.Error(t, err)
}

func TestDoEdit_NegativeIndex(t *testing.T) {

	table := newTestTable()

	err := table.DoEdit(map[string]any{"name": "nope"}, -1)

	require.Error(t, err)
}

func TestDoEdit_IndexTooLarge(t *testing.T) {

	table := newTestTable()

	err := table.DoEdit(map[string]any{"name": "nope"}, 99)

	require.Error(t, err)
}

func TestDoEdit_EditNotAllowed(t *testing.T) {

	table := newTestTable()
	table.CanEdit = false

	err := table.DoEdit(map[string]any{"name": "nope"}, 0) // editing an existing row

	require.Error(t, err)
}

/******************************************
 * DoEdit() - Missing / Nil Data Keys
 *
 * DoEdit pulls each Form field's value with data[field.Path].  A key the caller
 * omits therefore reads back as nil, which is passed straight to Schema.Set.  How
 * that nil is handled depends entirely on the field's schema type, so these tests
 * pin the (sometimes surprising) behavior at the boundary.
 ******************************************/

// A missing key for an UNCONSTRAINED string field is not an error: Set(nil) writes a
// nil, silently clearing the column.  Callers that want to preserve untouched fields
// must supply every Form field's current value in `data`.
func TestDoEdit_MissingStringKeyClearsField(t *testing.T) {

	table := newTestTable() // testSchema(): name/age are unconstrained
	db := table.Object.(*testDatabase)

	// Omit "name" entirely; supply a valid "age".
	err := table.DoEdit(map[string]any{"age": 30}, 0)

	require.NoError(t, err)
	assert.Nil(t, db.Data[0]["name"]) // nil overwrote the previous "John Connor"
	assert.EqualValues(t, 30, db.Data[0]["age"])
}

// A missing key for an integer field IS an error: Set(nil) fails to coerce nil into
// an integer, so DoEdit aborts and the row is left untouched.
func TestDoEdit_MissingIntegerKeyFails(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	// Supply a valid "name" but omit "age".
	err := table.DoEdit(map[string]any{"name": "Kyle Reese"}, 0)

	require.Error(t, err)
	assert.EqualValues(t, 20, db.Data[0]["age"]) // age field never reached a valid write
}

// A completely nil data map is rejected, because the integer "age" field cannot be
// set from a nil value.
func TestDoEdit_NilDataMap(t *testing.T) {

	table := newTestTable()

	err := table.DoEdit(nil, 0)

	require.Error(t, err)
}

// When the schema marks a field Required, omitting its key is rejected by validation
// and the original row is left untouched.
func TestDoEdit_MissingRequiredKeyFails(t *testing.T) {

	table := newConstrainedTable() // "name" is Required
	db := table.Object.(*testDatabase)

	// Omit the Required "name"; supply a valid "age".
	err := table.DoEdit(map[string]any{"age": 30}, 0)

	require.Error(t, err)
	assert.Equal(t, "John Connor", db.Data[0]["name"]) // original row untouched
}

/******************************************
 * DoEdit() - Schema Validation on Set
 *
 * As of rosetta v0.26+, Schema.Set runs the schema's validation rules before
 * writing.  DoEdit therefore surfaces validation failures as errors, and stores
 * the rewritten (clamped/truncated) value when a rule rewrites rather than rejects.
 * The unconstrained schema in testSchema() never exercises this, so these tests
 * use a constrained schema instead.
 ******************************************/

// newConstrainedTable assembles a Table whose row schema constrains its fields:
// "name" is required and capped at 5 characters, and "age" is clamped to 0..150.
func newConstrainedTable() Table {
	s := schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"data": schema.Array{
					MinLength: 1,
					MaxLength: 6,
					Items: schema.Object{
						Properties: schema.ElementMap{
							"name": schema.String{Required: true, MaxLength: 5},
							"age":  schema.Integer{Minimum: null.NewInt64(0), Maximum: null.NewInt64(150)},
						},
					},
				},
			},
		},
	}
	f := testForm()
	return New(&s, &f, testData(), "data", testIconProvider{}, "http://localhost/table")
}

// A value that violates a "required" rule makes Schema.Set fail, so DoEdit aborts
// the whole row edit with an error.
func TestDoEdit_ValidationRejectsRequired(t *testing.T) {

	table := newConstrainedTable()
	db := table.Object.(*testDatabase)

	err := table.DoEdit(map[string]any{"name": "", "age": 30}, 0)

	require.Error(t, err)
	assert.Equal(t, "John Connor", db.Data[0]["name"]) // original row is left untouched
}

// A value that cannot be coerced to the field's type (a non-numeric integer) makes
// Schema.Set fail, so DoEdit returns an error rather than storing a zero.
func TestDoEdit_ValidationRejectsBadType(t *testing.T) {

	table := newConstrainedTable()

	err := table.DoEdit(map[string]any{"name": "Bob", "age": "not-a-number"}, 0)

	require.Error(t, err)
}

// When a rule REWRITES rather than rejects, Schema.Set stores the rewritten value:
// a name longer than MaxLength is truncated to fit.
func TestDoEdit_ValidationTruncatesString(t *testing.T) {

	table := newConstrainedTable()
	db := table.Object.(*testDatabase)

	err := table.DoEdit(map[string]any{"name": "Abcdefghij", "age": 30}, 0)

	require.NoError(t, err)
	assert.Equal(t, "Abcde", db.Data[0]["name"]) // truncated to MaxLength (5 runes)
}

// An integer above the schema's Maximum is clamped down to that maximum rather
// than rejected.
func TestDoEdit_ValidationClampsInteger(t *testing.T) {

	table := newConstrainedTable()
	db := table.Object.(*testDatabase)

	err := table.DoEdit(map[string]any{"name": "Bob", "age": 999}, 0)

	require.NoError(t, err)
	assert.EqualValues(t, 150, db.Data[0]["age"]) // clamped to Maximum
}

/******************************************
 * DoEdit() - Field Allow-listing (Mass-Assignment Guard)
 *
 * DoEdit writes only the fields it finds in the Form (via AllElements), never the
 * keys it finds in the caller-supplied `data`.  That makes the Form an allow-list:
 * a client cannot write a column that is not an editable Form field, no matter what
 * keys it injects into `data`.  These tests pin that invariant, since the only thing
 * enforcing it is AllElements()'s behavior -- which lives in the (recently upgraded)
 * form package.
 ******************************************/

// newAllowListTable assembles a Table whose row schema has a "secret" field that the
// caller might try to inject.  Whether "secret" appears in the Form (and how) is left
// to each test via the supplied Form children.
func newAllowListTable(children []form.Element) Table {
	s := schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"data": schema.Array{
					MinLength: 1,
					MaxLength: 6,
					Items: schema.Object{
						Properties: schema.ElementMap{
							"name":   schema.String{},
							"age":    schema.Integer{},
							"secret": schema.String{},
						},
					},
				},
			},
		},
	}
	f := form.Element{Type: "layout-vertical", Children: children}
	return New(&s, &f, testData(), "data", testIconProvider{}, "http://localhost/table")
}

// A key in `data` that exists in the schema but has no matching Form field is
// silently ignored -- DoEdit never writes it.
func TestDoEdit_IgnoresKeyNotInForm(t *testing.T) {

	// Form exposes only "name" and "age"; "secret" is absent from the Form.
	table := newAllowListTable([]form.Element{
		{Type: "text", Label: "Name", Path: "name"},
		{Type: "text", Label: "Age", Path: "age"},
	})
	db := table.Object.(*testDatabase)

	err := table.DoEdit(map[string]any{"name": "Bob", "age": 1, "secret": "INJECTED"}, 0)

	require.NoError(t, err)
	assert.Equal(t, "Bob", db.Data[0]["name"])
	assert.Nil(t, db.Data[0]["secret"]) // never written: not an editable Form field
}

// A ReadOnly Form field is omitted by AllElements, so a client cannot set it through
// DoEdit even though it appears in the Form.
func TestDoEdit_IgnoresReadOnlyField(t *testing.T) {

	// "secret" IS in the Form, but marked ReadOnly.
	table := newAllowListTable([]form.Element{
		{Type: "text", Label: "Name", Path: "name"},
		{Type: "text", Label: "Age", Path: "age"},
		{Type: "text", Label: "Secret", Path: "secret", ReadOnly: true},
	})
	db := table.Object.(*testDatabase)

	err := table.DoEdit(map[string]any{"name": "Bob", "age": 1, "secret": "INJECTED"}, 0)

	require.NoError(t, err)
	assert.Equal(t, "Bob", db.Data[0]["name"])
	assert.Nil(t, db.Data[0]["secret"]) // never written: ReadOnly fields are not editable
}

/******************************************
 * DoDelete()
 ******************************************/

func TestDoDelete_Success(t *testing.T) {

	table := newTestTable()
	db := table.Object.(*testDatabase)

	err := table.DoDelete(0)

	require.NoError(t, err)
	require.Equal(t, 1, len(db.Data))
	assert.Equal(t, "Sarah Connor", db.Data[0]["name"])
}

func TestDoDelete_NotAllowed(t *testing.T) {

	table := newTestTable()
	table.CanDelete = false
	db := table.Object.(*testDatabase)

	err := table.DoDelete(0)

	require.Error(t, err)
	assert.Equal(t, 2, len(db.Data)) // nothing removed
}

func TestDoDelete_IndexOutOfRange(t *testing.T) {

	table := newTestTable()

	// schema.Remove returns false for an out-of-range index => DoDelete returns an error
	err := table.DoDelete(99)

	require.Error(t, err)
}
