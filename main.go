package main

import (
	"github.com/tomdiekmann/icu/cmd"
)

var version = "dev"

func main() {
	cmd.Execute(version)
}
