// Not a real Go module — this file exists solely to stop the root module's
// `./...` package discovery from descending into mobile/node_modules, where
// at least one npm package (flatted) ships a stray .go file that would
// otherwise be silently picked up by go build/vet/golangci-lint at the repo
// root, purely as an accident of an unrelated JS dependency.
module adera/mobile-boundary

go 1.26
