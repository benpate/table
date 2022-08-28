package table

import (
	"bytes"
	"fmt"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

func ExampleTable() {

	// Define a data schema to retrieve and validate the data
	s := schema.New(schema.Array{
		MaxLength: null.NewInt(10),
		Items: schema.Object{
			Properties: schema.ElementMap{
				"name": schema.String{},
				"age":  schema.Integer{},
			},
		},
	})

	// Define a UI schema to render the form
	f := form.Element{
		Type: "table",
		Children: []form.Element{
			{Type: "text", Label: "Name", Path: "name"},
			{Type: "number", Label: "Age", Path: "age"},
		},
	}

	// Define some data to render
	data := []maps.Map{
		{"name": "John Connor", "age": 20},
		{"name": "Sarah Connor", "age": 45},
	}

	table := New(&s, &f, &data, "", false, "http://localhost/update-form")

	// Render the form
	var buffer bytes.Buffer

	if err := table.DrawView(&buffer); err != nil {
		fmt.Println(err)
	}

	// Output:
	fmt.Println(buffer.String())
}
