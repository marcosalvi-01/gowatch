package main

import (
	"gowatch/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
