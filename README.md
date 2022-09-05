# Table 🍽

## Inline editor grid component for Go and htmx.

This repository implements a table/grid control with inline editing.  Tables are rendered on the server and swapped into the DOM via htmx `hx-get` and `hx-post` methods.

Tables work similar to other [form widgets](https://github.com/benpate/form), and require that you defined both a [data schema](https://github.com/benpate/rosetta/tree/main/schema) and a [UI schema](https://github.com/benpate/form) in order to render them.

```go
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
```

## DO NOT USE

This project is a work-in-progress, and should NOT be used by ANYONE, for ANY PURPOSE, under ANY CIRCUMSTANCES.  It is WILL BE CHANGED UNDERNEATH YOU WITHOUT NOTICE OR HESITATION, and is expressly GUARANTEED to blow up your computer, send your cat into an infinite loop, and combine your hot and cold laundry into a single cycle.

## Pull Requests Welcome

This library is a work in progress, and will benefit from your experience reports, use cases, and contributions.  If you have an idea for making this library better, send in a pull request.  We're all in this together! 🍽