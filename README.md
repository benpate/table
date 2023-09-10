# Table 🍽

[![GoDoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://pkg.go.dev/github.com/benpate/table)
[![Version](https://img.shields.io/github/v/release/benpate/table?include_prereleases&style=flat-square&color=brightgreen)](https://github.com/benpate/table/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/benpate/table/go.yml?style=flat-square)](https://github.com/benpate/table/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/table?style=flat-square)](https://goreportcard.com/report/github.com/benpate/table)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/table.svg?style=flat-square)](https://codecov.io/gh/benpate/table)

## Inline editor grid component for Go and htmx.

This repository implements a table/grid control with inline editing.  Tables are rendered on the server and swapped into the DOM via htmx `hx-get` and `hx-post` methods.

Tables work similar to other [form widgets](https://github.com/benpate/form), and require that you defined both a [data schema](https://github.com/benpate/rosetta/tree/main/schema) and a [UI schema](https://github.com/benpate/form) in order to render them.

```go
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
