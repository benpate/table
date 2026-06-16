package table

import (
	"net/url"
	"testing"

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

	f.Fuzz(func(t *testing.T, edit string, delete string) {

		table := newTestTable()

		params := &url.URL{RawQuery: url.Values{"edit": {edit}, "delete": {delete}}.Encode()}

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
