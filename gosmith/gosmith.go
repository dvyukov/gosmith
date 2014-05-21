package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
)

var (
	seed    = flag.Int64("seed", 0, "random generator seed")
	workdir = flag.String("dir", "", "directory to write the program to")
)

func main() {
	flag.Parse()
	if *workdir == "" {
		fmt.Fprintf(os.Stderr, "-dir flag is missing\n")
		os.Exit(1)
	}
	rand.Seed(*seed)
	writeProgram(*workdir)
}
