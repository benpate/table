# table — Notes for AI Agents

This package renders a server-side editable grid over htmx: columns are `form.Element`s, data is described by a rosetta schema, and every cell is drawn by the [benpate/form](https://github.com/benpate/form) widget registry. [README.md](README.md) shows the setup; [example/](example/) is a runnable demo.

## Rendering

- **A misconfigured table renders NOTHING, with only an error return to say why.** `Draw`/`DrawViewString` fail with a wrapped derp error, never a panic or partial page, so a mistake reads as "blank table". The two usual causes: an `Element.Type` that is not a registered form widget (call `form/widget.UseAll()` in every `main()` and `TestMain` — nothing registers it implicitly), and row data rosetta cannot index into. Unwrap the derp chain with `errors.Unwrap` in a loop; `Error()` shows only the outermost frame. Any `ExampleXxx` that renders a table needs an `// Output:` comment so CI catches a swallowed error.
- **Row data must live in a type rosetta can index into.** A plain `[]map[string]any` fails halfway: the top-level `Schema.Get` returns the slice and the header renders, then reading row "0" in `drawTable` fails with `Object must be a PointerGetter`. Use `sliceof.Object[mapof.Any]`, or a struct implementing `schema.PointerGetter` (see `testDatabase` in [table_test.go](table_test.go)).
- **`Path` must resolve to a `schema.Array` in the schema.** `getTableElement` in [table.go](table.go) errors otherwise. The array's `MaxLength` gates the Add control and `MinLength` gates Delete; `drawTable` computes those as per-render locals so bounds never mutate the caller's `CanAdd`/`CanDelete`.
- **There is no `"number"` widget.** Use `"text"` for numeric columns — it reads the input type from the data schema, so a `schema.Integer` column still renders `<input type="number">`.
- **The `Allow*`/`UseLookupProvider` builders are value receivers that return a copy.** `t.AllowNone()` alone does nothing; you must use the returned value. `New` defaults all three permissions to true.

## The edit round trip

- **`Do` and `Draw` both read the row index from query parameters (`?edit=N`, `?delete=N`, `?add=true`), so both ends must be driven by the same `getURL`-style links.** `Draw` bounds-checks: an out-of-range `edit` falls back to view-only, and `focus` is clamped to a valid column. `DoEdit` treats `editIndex == length` as "add a new row" (gated by `CanAdd`); anything larger is an error.
- **`DoEdit` is not atomic — on error the caller MUST discard the object.** It writes field-by-field through `Schema.Set`, which validates each value as it writes (and can clamp, truncate, or reject), so a later field failing leaves earlier fields already written into `widget.Object`. That is the documented contract in [table_do.go](table_do.go): never persist an object after `Do`/`DoEdit` returned an error.
- **`DoEdit` only writes fields present in the Form — that is the mass-assignment guard.** It iterates `widget.Form.AllElements()`, which also prunes `ReadOnly` elements, so extra keys in the posted data are silently ignored and read-only columns cannot be set. Do not "optimize" the loop to iterate the posted map instead.

## HTML conventions

- **Icon markup is injected raw via `InnerHTML` — the `IconProvider` is a trust boundary.** The table asks it for `plus`, `save`, `cancel`, `edit`, and `delete`; the interface is satisfied structurally (Emissary passes `factory.Icons()`). Never route end-user data through an IconProvider.
- **Cell values are escaped by the form widgets, not by this package.** Each cell calls `field.View`/`field.Edit` on the row value; the escaping contract lives in the form package (values via `InnerText`, definitions trusted). See form's AGENTS.md before adding any direct `b.WriteString` here.
- **The package ships no CSS.** It emits the class vocabulary `grid`, `grid-header`, `grid-row`, `grid-cell`, `grid-editable`, `grid-controls`, `hover-trigger`, plus `link` and `text-green` on controls; consumers style them (Emissary does in its theme-global stylesheet). Renaming a class is a breaking change for every consumer's theme.
- **`focusField` clones the Options map before setting `focus` — keep it that way.** The Form definition is shared across rows and renders; mutating a child's `Options` in place would leak the focus flag into later rows.
- **The `nolint:scopeguard` on `b.TD()` calls is important.** Opening the cell is a side effect that must run unconditionally; moving it inside the adjacent `if` (as the linter suggests) drops the cell for columns without that option.

## A blank table is an error return, not a panic — check these two causes first

`Draw`/`DrawViewString` report a misconfiguration by returning a wrapped derp error, so a broken table renders as an empty page and can survive that way for a long time. The README example, the package `ExampleTable`, and the runnable `example/` server were all three broken this way at once. Two causes account for both:

- **An unregistered widget `Type`.** `form.Element.Type` must name a widget registered through `form/widget.UseAll()` (or an explicit `form.Use`), and **there is no `"number"` widget** — an unknown name fails with `form.Widget: Unrecognized form widget`. Use `"text"`, which reads the input type from the *data* schema, so a `schema.Integer` column still renders `<input type="number" step="1">`. Any `main()` that draws forms must call `widget.UseAll()` itself; nothing registers it implicitly.

- **Row data rosetta cannot index into.** A plain `[]map[string]any` is not usable as table data. The top-level `Schema.Get` succeeds (it returns the slice) and the row count is right, so the *header* renders and then reading row `0` fails with `schema.getProperty_PointerOnly: Object must be a PointerGetter`. Use `sliceof.Object[mapof.Any]`, or a struct implementing `schema.PointerGetter`.

Neither the compiler nor any go-vet gate catches either one, and `fmt.Println(table.DrawViewString())` swallows the error into the output. **Every `ExampleXxx` that renders a form or table needs an `// Output:` comment** — that alone would have caught all three. When a table draws blank, unwrap the derp chain in a loop; `Error()` shows only the outermost `location: message`.
