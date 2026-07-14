package main

import (
	_ "fixture/cmd/tinyidp-xapp/internal/runtime"
	_ "fixture/internal/authn" // want `embedding application imports private package "fixture/internal/authn"`
)

func main() {}
