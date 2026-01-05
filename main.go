package main

import (
	"github.com/marcosalvi-01/gowatch/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
