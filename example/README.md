# table/example

A minimal, runnable HTTP server that demonstrates the [`table`](../README.md) widget end-to-end: it renders an editable task grid and applies add/edit/delete actions to an in-memory `Database`. Use it as a reference for wiring a `table.Table` into a real handler.

```sh
cd example   # required -- see below
go run .
# open http://localhost:8080/index.html
```

## What matters here

- **Run it from inside this directory.** `getFile` loads `index.html` with `os.DirFS(".")`, which resolves against the *current working directory*, not the binary's location. Launching from the repo root serves a 500 because the file isn't found. `cd example` first.

- **`Database` is the load-bearing seam, not the slice.** `table` reads and writes data through the `schema.PointerGetter` interface, so the example's `Database.GetPointer("data")` returns `&d.Data` (a pointer) — returning the value would make edits no-ops. Any host object passed to `table.New` must implement `GetPointer` the same way.

- **The handler is the canonical GET/POST split.** `handleTable` shows the intended contract: GET → `Draw(r.URL, w)` (router reads `add`/`edit`/`focus` query params); POST → `Do(r.URL, postData)` then `DrawView`. A persistent store would add a `db.Save()` between `Do` and the redraw — the comment marks the spot.

- **`bind` is a stand-in for your framework.** It flattens `r.Form` to `map[string]any` taking the first value per key. Real apps usually let echo/gin/etc. do this; it exists here only to keep the demo dependency-free.

- **`IconProvider` uses Bootstrap Icons.** The returned `<i class="bi ...">` markup assumes the Bootstrap Icons CSS is loaded (see `index.html`). Swap this implementation to use any icon set; `table` only calls `Get`/`Write`.

This is a `package main` demo, intentionally hacky (it says so in the comments). It is not imported by the library and is excluded from Sonar analysis.
