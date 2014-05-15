package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	parallelism = flag.Int("p", runtime.NumCPU(), "number of parallel tests")
	workDir     = flag.String("workdir", "./work", "working directory for temp files")

	gosmithBin string
	ssadumpBin string

	statTotal   uint64
	statBuild   uint64
	statSsadump uint64
	statKnown   uint64

	knownBuildBugs = []*regexp.Regexp{
		regexp.MustCompile("internal compiler error: out of fixed registers"),
		regexp.MustCompile("constant [0-9]* overflows"),
	}

	knownSsadumpBugs = []*regexp.Regexp{
		regexp.MustCompile("constant .* overflows"),
	}
)

func main() {
	flag.Parse()
	gosmithBin = buildBinary("code.google.com/p/gosmith/gosmith", "gosmith")
	ssadumpBin = buildBinary("code.google.com/p/go.tools/cmd/ssadump", "ssadump")
	log.Printf("testing with %v workers", *parallelism)
	os.MkdirAll(filepath.Join(*workDir, "tmp"), os.ModePerm)
	os.MkdirAll(filepath.Join(*workDir, "bug"), os.ModePerm)
	rand.Seed(time.Now().UnixNano())
	seed := rand.Int63()
	for p := 0; p < *parallelism; p++ {
		go func() {
			for {
				s := atomic.AddInt64(&seed, 1)
				test(fmt.Sprintf("%v", s))
				atomic.AddUint64(&statTotal, 1)
			}
		}()
	}
	for {
		time.Sleep(3 * time.Second)
		total := atomic.LoadUint64(&statTotal)
		build := atomic.LoadUint64(&statBuild)
		known := atomic.LoadUint64(&statKnown)
		ssadump := atomic.LoadUint64(&statSsadump)
		log.Printf("%v tests, %v known bugs, %v build failures, %v ssadump failures",
			total, known, build, ssadump)
	}
}

func buildBinary(pkg, prog string) string {
	bin := filepath.Join(*workDir, prog)
	out, err := exec.Command("go", "build", "-o", bin, pkg).CombinedOutput()
	if err != nil {
		log.Fatalf("failed to build %v: %v\n%v\n", pkg, err, string(out))
	}
	return bin
}

func test(seed string) {
	path := filepath.Join(*workDir, "tmp", seed)
	src := filepath.Join(path, "src.go")
	os.Mkdir(path, os.ModePerm)
	keep := false
	defer func() {
		if keep {
			os.Rename(path, filepath.Join(*workDir, "bug", seed))
		} else {
			os.RemoveAll(path)
		}
	}()
	ok := generateSource(src, seed)
	if !ok {
		return
	}
	if testBuild(src, seed, false) {
		keep = true
		return
	}
	if testBuild(src, seed, true) {
		keep = true
		return
	}
	if testSsadump(src, seed) {
		keep = true
		return
	}
	//- gofmt idempotentness (gofmt several times and compare results)
	//- ast/types/ssa robustness and idempotentness (generate, serialize, parse, serialize, parse, compare).
	//- govet robustness
	//- compare gc, gccgo, ssa/interp output
}

func generateSource(src, seed string) bool {
	srcf, err := os.Create(src)
	defer func() {
		srcf.Close()
	}()
	if err != nil {
		log.Printf("failed to create source file: %v", err)
		return false
	}
	out, err := exec.Command(gosmithBin, "-seed", seed).CombinedOutput()
	if err != nil {
		log.Printf("failed to execute gosmith for seed %v: %v\n%v\n", seed, err, string(out))
		return false
	}
	srcf.Write(out)
	return true
}

func testBuild(src string, seed string, race bool) bool {
	args := []string{"build", "-o", src + ".bin"}
	if race {
		args = append(args, "-race")
	}
	args = append(args, src)
	out, err := exec.Command("go", args...).CombinedOutput()
	if err == nil {
		return false
	}
	for _, known := range knownBuildBugs {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	outname := src + ".build"
	if race {
		outname += ".race"
	}
	outf, err := os.Create(outname)
	if err != nil {
		log.Printf("failed to create output file: %v", err)
	} else {
		outf.Write(out)
		outf.Close()
	}
	log.Printf("build failed, seed %v\n", seed)
	atomic.AddUint64(&statBuild, 1)
	return true
}

func testSsadump(src string, seed string) bool {
	out, err := exec.Command(ssadumpBin, "-build=CDPF", src).CombinedOutput()
	if err == nil {
		return false
	}
	for _, known := range knownSsadumpBugs {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	outf, err := os.Create(src + ".ssadump")
	if err != nil {
		log.Printf("failed to create output file: %v", err)
	} else {
		outf.Write(out)
		outf.Close()
	}
	log.Printf("ssadump failed, seed %v\n", seed)
	atomic.AddUint64(&statSsadump, 1)
	return true
}
