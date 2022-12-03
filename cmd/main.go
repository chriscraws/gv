package main

import (
	"fmt"
	"os"

	"github.com/chriscraws/gv"
)

func main() {
	// Arguments
	if len(os.Args) != 3 {
		fmt.Println("usage: gv <main_package_path> <output_path>")
		return
	}
	mainPkgPath := os.Args[1]
	outputPath := os.Args[2]

	c := gv.Compiler{MainPkgPath: mainPkgPath}

	spv, err := c.Compile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "compilation failed: %s", err)
	}

	os.WriteFile(outputPath, spv, 0644)
}
