package main

import (
	"atlas-refactor/internal/webslice"
	"fmt"
	"os"
)

func main() {
	if e := webslice.Run(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
