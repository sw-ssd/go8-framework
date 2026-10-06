// Package e2e holds the web+API end-to-end test.
//
// The actual test file is guarded by the `e2e` build tag so that the normal
// `go test ./...` suite (and `go build ./...`) does not pull in Docker/Playwright
// dependencies. Run it explicitly with:
//
//	go test -tags e2e ./e2e
//
// which requires Docker (for the Postgres and browser containers).
package e2e
