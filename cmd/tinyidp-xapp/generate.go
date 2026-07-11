package main

// Generate the checked-in TypeScript declarations and importable runtime
// package from the product specification. Generation runs from the repository
// root because xgoja artifact outputs are root-relative.

//go:generate sh -c "cd ../.. && go run ../go-go-goja/cmd/xgoja gen-dts -f cmd/tinyidp-xapp/xgoja.yaml"
//go:generate sh -c "cd ../.. && go run ../go-go-goja/cmd/xgoja generate -f cmd/tinyidp-xapp/xgoja.yaml"
