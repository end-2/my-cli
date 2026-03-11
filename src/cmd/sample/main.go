package main

import (
	"flag"
	"fmt"
)

var (
	Version = "dev"
)

func main() {
	showVersion := flag.Bool("version", false, "print binary version")
	flag.Parse()

	if *showVersion {
		fmt.Println(Version)
		return
	}

	fmt.Println("Hello MY CLI")
}
