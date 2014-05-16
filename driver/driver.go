package main

/*
Usage instructions:
hg sync
hg clpatch 93420045
export ASAN_OPTIONS="detect_leaks=0"
CC=clang CFLAGS="-fsanitize=address -fno-omit-frame-pointer -fno-common -O2" ./make.bash
CC=clang CFLAGS="-fsanitize=address -fno-omit-frame-pointer -fno-common -O2" GOARCH=386 go tool dist bootstrap
CC=clang CFLAGS="-fsanitize=address -fno-omit-frame-pointer -fno-common -O2" GOARCH=arm go tool dist bootstrap
go install -race -a std
go install -a std
go get -u code.google.com/p/gosmith/gosmith
go get -u code.google.com/p/go.tools/cmd/ssadump
go run driver.go
*/

import (
	"bytes"
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

	statTotal   uint64
	statBuild   uint64
	statSsadump uint64
	statGofmt   uint64
	statKnown   uint64

	knownBuildBugs = []*regexp.Regexp{
		// gc
		regexp.MustCompile("internal compiler error: out of fixed registers"),
		regexp.MustCompile("constant [0-9]* overflows"),
		//regexp.MustCompile("cannot use _ as value"),
		regexp.MustCompile("internal compiler error: walkexpr ORECV"),

		// gccgo
		regexp.MustCompile("internal compiler error: in fold_convert_loc, at fold-const.c:2072"),
		regexp.MustCompile("internal compiler error: in fold_binary_loc, at fold-const.c:10024"),
	}

	knownSsadumpBugs = []*regexp.Regexp{
		regexp.MustCompile("constant .* overflows"),
	}
)

func main() {
	flag.Parse()
	log.Printf("testing with %v workers", *parallelism)
	os.Setenv("ASAN_OPTIONS", "detect_leaks=0 detect_odr_violation=2 detect_stack_use_after_return=1")
	os.MkdirAll(filepath.Join(*workDir, "tmp"), os.ModePerm)
	os.MkdirAll(filepath.Join(*workDir, "bug"), os.ModePerm)
	rand.Seed(time.Now().UnixNano())
	seed := rand.Int63()
	for p := 0; p < *parallelism; p++ {
		go func() {
			for {
				s := atomic.AddInt64(&seed, 1)
				t := &Test{seed: fmt.Sprintf("%v", s)}
				t.Do()
				atomic.AddUint64(&statTotal, 1)
			}
		}()
	}
	for {
		total := atomic.LoadUint64(&statTotal)
		build := atomic.LoadUint64(&statBuild)
		known := atomic.LoadUint64(&statKnown)
		ssadump := atomic.LoadUint64(&statSsadump)
		gofmt := atomic.LoadUint64(&statGofmt)
		log.Printf("%v tests, %v known, %v build, %v ssadump, %v gofmt",
			total, known, build, ssadump, gofmt)
		time.Sleep(5 * time.Second)
	}
}

type Test struct {
	seed   string
	path   string
	src    string
	keep   bool
	srcbuf []byte
}

func (t *Test) Do() {
	t.path = filepath.Join(*workDir, "tmp", t.seed)
	os.Mkdir(t.path, os.ModePerm)
	defer func() {
		if t.keep {
			os.Rename(t.path, filepath.Join(*workDir, "bug", t.seed))
		} else {
			os.RemoveAll(t.path)
		}
	}()
	if !t.generateSource() {
		return
	}
	if t.Build("gc", "amd64", false) {
		t.keep = true
		return
	}
	if t.Build("gc", "386", false) {
		t.keep = true
		return
	}
	if t.Build("gc", "arm", false) {
		t.keep = true
		return
	}
	if t.Build("gc", "amd64", true) {
		t.keep = true
		return
	}
	if t.Build("gccgo", "amd64", false) {
		t.keep = true
		return
	}
	if t.Ssadump() {
		t.keep = true
		return
	}
	//- gofmt idempotentness (gofmt several times and compare results)
}

func (t *Test) generateSource() bool {
	t.src = filepath.Join(t.path, "src.go")
	srcf, err := os.Create(t.src)
	defer func() {
		srcf.Close()
	}()
	if err != nil {
		log.Printf("failed to create source file: %v", err)
		return false
	}
	t.srcbuf, err = exec.Command("gosmith", "-seed", t.seed).CombinedOutput()
	if err != nil {
		log.Printf("failed to execute gosmith for seed %v: %v\n%v\n", t.seed, err, string(t.srcbuf))
		return false
	}
	srcf.Write(t.srcbuf)
	return true
}

func (t *Test) Build(compiler, goarch string, race bool) bool {
	args := []string{"build", "-o", t.src + "." + goarch, "-compiler", compiler}
	if race {
		args = append(args, "-race")
	}
	args = append(args, t.src)
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOARCH="+goarch)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return false
	}
	for _, known := range knownBuildBugs {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	typ := compiler + "." + goarch
	if race {
		typ += ".race"
	}
	outname := t.src + ".build." + typ
	outf, err := os.Create(outname)
	if err != nil {
		log.Printf("failed to create output file: %v", err)
	} else {
		outf.Write(out)
		outf.Close()
	}
	log.Printf("%v build failed, seed %v\n", typ, t.seed)
	atomic.AddUint64(&statBuild, 1)
	return true
}

func (t *Test) Ssadump() bool {
	out, err := exec.Command("ssadump", "-build=CDPF", t.src).CombinedOutput()
	if err == nil {
		return false
	}
	for _, known := range knownSsadumpBugs {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	outf, err := os.Create(t.src + ".ssadump")
	if err != nil {
		log.Printf("failed to create output file: %v", err)
	} else {
		outf.Write(out)
		outf.Close()
	}
	log.Printf("ssadump failed, seed %v\n", t.seed)
	atomic.AddUint64(&statSsadump, 1)
	return true
}

func (t *Test) Gofmt() bool {
	formatted, err := exec.Command("gofmt", t.src).CombinedOutput()
	if err != nil {
		outf, err := os.Create(t.src + ".gofmt")
		if err != nil {
			log.Printf("failed to create output file: %v", err)
		} else {
			outf.Write(formatted)
			outf.Close()
		}
		log.Printf("gofmt failed, seed %v\n", t.seed)
		atomic.AddUint64(&statGofmt, 1)
		return true
	}
	fname := t.src + ".formatted"
	outf, err := os.Create(fname)
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return false
	}
	outf.Write(formatted)
	outf.Close()

	formatted2, err := exec.Command("gofmt", fname).CombinedOutput()
	if err != nil {
		outf, err := os.Create(t.src + ".gofmt")
		if err != nil {
			log.Printf("failed to create output file: %v", err)
		} else {
			outf.Write(formatted2)
			outf.Close()
		}
		log.Printf("gofmt failed, seed %v\n", t.seed)
		atomic.AddUint64(&statGofmt, 1)
		return true
	}
	outf2, err := os.Create(t.src + ".formatted2")
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return false
	}
	outf2.Write(formatted2)
	outf2.Close()

	if bytes.Compare(formatted, formatted2) != 0 {
		log.Printf("nonidempotent gofmt, seed %v\n", t.seed)
		atomic.AddUint64(&statGofmt, 1)
		return true
	}

	removeWs := func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' {
			return -1
		}
		return r
	}
	stripped := bytes.Map(removeWs, t.srcbuf)
	stripped2 := bytes.Map(removeWs, formatted)
	if bytes.Compare(stripped, stripped2) != 0 {
		log.Printf("nonidempotent gofmt, seed %v\n", t.seed)
		atomic.AddUint64(&statGofmt, 1)
		return true
	}

	return false
}
