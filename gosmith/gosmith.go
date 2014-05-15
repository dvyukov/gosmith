package main

import (
	"bufio"
	"flag"
	"math/rand"
	"os"
)

var (
	seed           = flag.Int64("seed", 0, "random generator seed")
	incorrect      = flag.Bool("incorrect", false, "generate incorrect programs")
	nonterminating = flag.Bool("nonterminating", false, "generate nonterminating programs")
)

func main() {
	flag.Parse()
	if *incorrect {
		*nonterminating = true
	}
	rand.Seed(*seed)
	w := bufio.NewWriter(os.Stdout)
	c := NewContext(w, *incorrect, *nonterminating)
	c.program()
	w.Flush()
}
