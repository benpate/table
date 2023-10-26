package main

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/table"
)

// database is a hacky data structure where we'll store the example data
var database Database

// main initializes the app and starts an HTTP server.
// After you run this program, you can view the demo at http://localhost:8080
func main() {

	// Initialize the database
	database = getDefaultTableData()

	// Define HTTP routes
	mux := http.NewServeMux()
	mux.Handle("/index.html", getFile("index.html"))
	mux.Handle("/table", handleTable())

	// Start the HTTP server
	fmt.Println("Listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

/******************************************
 * HTTP Handlers
 ******************************************/

// getFile is an http.HandlerFunc that uses a static file in the local directory
// as a go Template, and serves this file as the response.
func getFile(filename string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// Read the file from the filesystem
		// (super inefficient, but works great for a quick example)
		file, err := fs.ReadFile(os.DirFS("."), filename)

		// Handle errors (boo!)
		if err != nil {
			writeError(w, err)
			return
		}

		// Write the file to the response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(file)
	}
}

// handleTable is an HTTP handler that displays the table widget`
// and updates the database when the user makes changes.
func handleTable() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// Get the table widget
		exampleTable := getTable()

		// If this is a GET request, then simply draw the table
		if r.Method == http.MethodGet {

			if err := exampleTable.Draw(r.URL, w); err != nil {
				writeError(w, err)
				return
			}
			return
		}

		// Collect the data from the POST request
		// Usually, echo (or other tools) does this for me.
		postData, err := bind(r)

		if err != nil {
			writeError(w, err)
			return
		}

		// Apply the changes to the database
		if err := exampleTable.Do(r.URL, postData); err != nil {
			writeError(w, err)
			return
		}

		// If we weren't using an in-memory data structure,
		// there would probably be some sort of db.Save() call here.

		// Finally, redraw the table
		_ = exampleTable.DrawView(w)
	}
}

/******************************************
 * Data Structures for the Demo Table
 ******************************************/

// getTable defines the Table widget for this example.
// https://github.com/benpate/table
func getTable() table.Table {

	schema := getTableSchema()
	form := getTableForm()

	return table.New(
		&schema,
		&form,
		&database,
		"data",
		IconProvider{},
		"/table",
	)
}

// getTableSchema defines the data layout for this example.
// https://github.com/benpate/rosetta/schema
func getTableSchema() schema.Schema {

	return schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"data": schema.Array{
					MinLength: 1,
					MaxLength: 6,
					Items: schema.Object{
						Properties: schema.ElementMap{
							"taskId":      schema.String{},
							"label":       schema.String{MaxLength: 128},
							"description": schema.String{MaxLength: 1024},
							"status":      schema.String{Enum: []string{"New", "Pending", "Waiting", "In Progress", "Complete"}},
							"assignedTo":  schema.String{Enum: []string{"Alice", "Bob", "Carl", "Dave"}},
						},
					},
				},
			},
		},
	}
}

// getTableForm defines the UI layout for this example.
// https://github.com/benpate/form
func getTableForm() form.Element {

	return form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{
				Type:  "text",
				Path:  "label",
				Label: "Task Name",
			},
			{
				Type:  "textarea",
				Path:  "description",
				Label: "Description",
			},
			{
				Type:  "select",
				Path:  "status",
				Label: "Status",
			},
			{
				Type:  "select",
				Path:  "assignedTo",
				Label: "Assigned To",
			},
		},
	}
}

// getDefaultTableData defines the initial data for this example.
func getDefaultTableData() Database {
	return Database{
		Data: sliceof.Object[mapof.Any]{
			mapof.Any{
				"taskId":      "1",
				"label":       "Grocery Store",
				"description": "Some gibberish.",
				"status":      "In Progress",
				"assignedTo":  "Bob",
			},
			mapof.Any{
				"taskId":      "2",
				"label":       "Hardware Store",
				"description": "More gibberish here.",
				"status":      "Pending",
				"assignedTo":  "Alice",
			},
		},
	}
}

/******************************************
 * IconProvider
 ******************************************/

// IconProvider returns HTML code for displaying various icons.
// This helps separate the icon library from the table widget.
type IconProvider struct {
}

func (i IconProvider) Get(name string) string {

	switch name {

	case "plus": // plus sign
		return `<i class="bi bi-plus-circle-fill"></i>`

	case "edit": // pencil
		return `<i class="bi bi-pencil-square"></i>`

	case "delete": // trash can
		return `<i class="bi bi-trash"></i>`

	case "save": // checkmark
		return `<i class="bi bi-check-circle-fill"></i>`

	case "cancel": // X
		return `<i class="bi bi-x-circle-fill"></i>`
	}

	return name
}

func (i IconProvider) Write(name string, writer io.Writer) {
	_, _ = writer.Write([]byte(i.Get(name)))
}

/******************************************
 * Data Structure
 ******************************************/

// Database is a simple data structure that holds the example data.
// For it to work in the table.Table widget, it needs to implement the
// schema.PointerGetter interface
type Database struct {
	Data sliceof.Object[mapof.Any]
}

// GetPointer implements the schema.PointerGetter interface
func (d *Database) GetPointer(name string) (any, bool) {

	switch name {
	case "data":
		return &d.Data, true
	}

	return nil, false
}

/******************************************
 * Miscellaneous Helpers
 ******************************************/

// bind collects all of the form data from an HTTP request
// I'm used to a framework (like echo) doing this for me.
func bind(r *http.Request) (map[string]any, error) {

	result := make(map[string]any)

	if err := r.ParseForm(); err != nil {
		return result, err
	}

	for key, value := range r.Form {
		result[key] = value[0]
	}

	return result, nil
}

// writeError writes an error to the http.ResponseWriter.
// This is just some sugar to make the examples more readable.
func writeError(writer http.ResponseWriter, err error) {
	writer.WriteHeader(http.StatusInternalServerError)
	_, _ = writer.Write([]byte(err.Error()))
	derp.Report(err)
}
