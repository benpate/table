package table

import (
	"fmt"

	"github.com/benpate/form"
	"github.com/benpate/icon/bootstrap"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

func ExampleTable() {

	// Data schema defines the layout of the data.
	s := schema.New(schema.Array{
		MaxLength: null.NewInt(10),
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
	data := []maps.Map{
		{"name": "John Connor", "age": 20},
		{"name": "Sarah Connor", "age": 45},
	}

	// Create the new table and render it in HTML
	table := New(&s, &f, &data, "", bootstrap.Provider{}, "http://localhost/update-form")
	fmt.Println(table.DrawViewString())
}
