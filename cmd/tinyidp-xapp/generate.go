package main

// Generate the checked-in TypeScript declarations and importable runtime
// package from the product specification. Generation runs from the repository
// root because xgoja artifact outputs are root-relative. The generator is
// resolved from go.mod, rather than a sibling checkout, so a clean release
// checkout can reproduce the generated host.

//go:generate sh -c "cd ../.. && pnpm --dir cmd/tinyidp-xapp/app/frontend run build"
//go:generate sh -c "cd ../.. && go run github.com/go-go-golems/go-go-goja/cmd/xgoja gen-dts -f cmd/tinyidp-xapp/xgoja.yaml"
//go:generate sh -c "cd ../.. && go run github.com/go-go-golems/go-go-goja/cmd/xgoja generate -f cmd/tinyidp-xapp/xgoja.yaml --clean"
