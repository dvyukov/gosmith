package main

/*
Usage instructions:
hg sync
hg clpatch 93420045
export ASAN_OPTIONS="detect_leaks=0"
CC=clang CFLAGS="-fsanitize=address -fno-omit-frame-pointer -fno-common -O2" ./make.bash
CC=clang CFLAGS="-fsanitize=address -fno-omit-frame-pointer -fno-common -O2" GOARCH=386 go tool dist bootstrap
CC=clang CFLAGS="-fsanitize=address -fno-omit-frame-pointer -fno-common -O2" GOARCH=arm go tool dist bootstrap
GOARCH=386 go install std
GOARCH=arm go install std
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
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	parallelism = flag.Int("p", runtime.NumCPU(), "number of parallel tests")
	checkers    = flag.String("checkers", "all", "comma-delimited list of checkers (amd64,386,arm,nacl64,nacl32,race,gccgo,ssa,gofmt,exec)")
	workDir     = flag.String("workdir", "./work", "working directory for temp files")
	timeout     = flag.Int64("timeout", 10, "task timeout in seconds")

	statTotal   uint64
	statBuild   uint64
	statSsadump uint64
	statGofmt   uint64
	statExec    uint64
	statKnown   uint64

	knownBuildBugs   = make(map[string][]*regexp.Regexp)
	knownSsadumpBugs = []*regexp.Regexp{}
	knownExecBugs    = []*regexp.Regexp{
		regexp.MustCompile("^panic: "),
		regexp.MustCompile("go of nil func value"),
		regexp.MustCompile("panic: runtime error: index out of range"),
		regexp.MustCompile("panic: runtime error: slice bounds out of range"),
		regexp.MustCompile("panic: runtime error: invalid memory address or nil pointer dereference"),
		regexp.MustCompile("fatal error: all goroutines are asleep - deadlock!"),
		regexp.MustCompile("SIGABRT: abort"), // gc
		regexp.MustCompile("Aborted"),        // gccgo
		// nacl:
		regexp.MustCompile("Signal 6 from trusted code"),
		regexp.MustCompile("Signal 11 from trusted code"),
		regexp.MustCompile("Signal 6 from untrusted code"),
		regexp.MustCompile("Signal 11 from untrusted code"),
		regexp.MustCompile("fatal error: runtime: address space conflict"), // nacl says this when we exhaust all memory
		// bad:
		regexp.MustCompile("fatal error: slice capacity smaller than length"),
		regexp.MustCompile("copyabletopsegment"),
		regexp.MustCompile("scanbitvector"),
		regexp.MustCompile("runtime.gostartcallfn"),
		regexp.MustCompile("__go_map_delete"),                       // gccgo
		regexp.MustCompile("fatal error: runtime_lock: lock count"), // gccgo
		regexp.MustCompile("fatal error: stopm holding locks"),      // gccgo
		// gllgo:
		regexp.MustCompile("unexpected fault address 0x0"),
		// ssa interp:
		regexp.MustCompile("ssa/interp\\.\\(\\*frame\\)\\.runDefers"),
	}
)

func init() {
	knownBuildBugs["all"] = []*regexp.Regexp{
		regexp.MustCompile("constant .* overflows"), // problem is gosmith
	}

	knownBuildBugs["gc"] = []*regexp.Regexp{
		regexp.MustCompile("fallthrough statement out of place"),            // https://code.google.com/p/go/issues/detail?id=8041
		regexp.MustCompile("cannot take the address of"),                    // https://code.google.com/p/go/issues/detail?id=8074
		regexp.MustCompile("mixture of field:value and value initializers"), // https://code.google.com/p/go/issues/detail?id=8099
		regexp.MustCompile("sinit.c:1060 anylit"),                           // https://code.google.com/p/go/issues/detail?id=8099 (under asan)
		regexp.MustCompile("out of fixed registers"),                        // https://code.google.com/p/go/issues/detail?id=8025, https://code.google.com/p/go/issues/detail?id=8012
		regexp.MustCompile("gen_as_init"),                                   // https://code.google.com/p/go/issues/detail?id=8058
		//regexp.MustCompile("SIGABRT: abort"),                 // https://code.google.com/p/go/issues/detail?id=8076
	}
	knownBuildBugs["gc.amd64"] = []*regexp.Regexp{}
	knownBuildBugs["gc.386"] = []*regexp.Regexp{}
	knownBuildBugs["gc.arm"] = []*regexp.Regexp{}
	knownBuildBugs["gc.amd64.race"] = []*regexp.Regexp{}
	knownBuildBugs["gccgo"] = []*regexp.Regexp{
		regexp.MustCompile("internal compiler error: in fold_binary_loc, at fold-const.c:10024"),
		regexp.MustCompile("internal compiler error: in write_specific_type_functions, at go/gofrontend/types.cc:1819"),
		regexp.MustCompile("internal compiler error: in fold_convert_loc, at fold-const.c:2072"),
		regexp.MustCompile("internal compiler error: in do_determine_types, at go/gofrontend/statements.cc:400"),
		regexp.MustCompile("internal compiler error: verify_gimple failed"),
		regexp.MustCompile("internal compiler error: in descriptor, at go/gofrontend/gogo.cc:4572"),         // https://gcc.gnu.org/bugzilla/show_bug.cgi?id=61307
		regexp.MustCompile("internal compiler error: in check_bounds, at go/gofrontend/expressions.cc:480"), // https://gcc.gnu.org/bugzilla/show_bug.cgi?id=61308
		regexp.MustCompile("error: too many arguments"),
		regexp.MustCompile("error: expected '<-' or '='"),
		regexp.MustCompile("error: slice end must be integer"),
		regexp.MustCompile("error: argument 2 has incompatible type"),
		regexp.MustCompile("error: incompatible types in assignment (multiple-value function call in single-value context)"), // https://gcc.gnu.org/bugzilla/show_bug.cgi?id=61316
		regexp.MustCompile("__normal_iterator"),
		regexp.MustCompile("Unsafe_type_conversion_expression::do_get_backend"),

		// gllgo
		regexp.MustCompile("_Cfunc_LLVMTargetMachineEmitToMemoryBuffer"), // https://github.com/go-llvm/llgo/issues/174
		regexp.MustCompile("panic: unimplemented conversion"),            // https://github.com/go-llvm/llgo/issues/176
	}
}

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
		exec := atomic.LoadUint64(&statExec)
		log.Printf("%v tests, %v known, %v build, %v ssadump, %v gofmt, %v exec",
			total, known, build, ssadump, gofmt, exec)
		time.Sleep(3 * time.Second)
	}
}

type Test struct {
	seed   string
	path   string
	gopath string
	keep   bool
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
	if enabled("amd64") && t.Build("gc", "", "amd64", false) {
		t.keep = true
		return
	}
	if enabled("amd64") && enabled("exec") && t.Exec("gc", "", "amd64", false) {
		t.keep = true
		return
	}
	if enabled("386") && t.Build("gc", "", "386", false) {
		t.keep = true
		return
	}
	if enabled("386") && enabled("exec") && t.Exec("gc", "", "386", false) {
		t.keep = true
		return
	}
	if enabled("arm") && t.Build("gc", "", "arm", false) {
		t.keep = true
		return
	}
	if enabled("nacl64") && t.Build("gc", "nacl", "amd64p32", false) {
		t.keep = true
		return
	}
	if enabled("nacl64") && enabled("exec") && t.Exec("gc", "nacl", "amd64p32", false) {
		t.keep = true
		return
	}
	if enabled("nacl32") && t.Build("gc", "nacl", "386", false) {
		t.keep = true
		return
	}
	if enabled("nacl32") && enabled("exec") && t.Exec("gc", "nacl", "386", false) {
		t.keep = true
		return
	}
	if enabled("race") && t.Build("gc", "", "amd64", true) {
		t.keep = true
		return
	}
	if enabled("race") && enabled("exec") && t.Exec("gc", "", "amd64", true) {
		t.keep = true
		return
	}
	if enabled("gccgo") && t.Build("gccgo", "", "amd64", false) {
		t.keep = true
		return
	}
	if enabled("gccgo") && enabled("exec") && t.Exec("gccgo", "", "amd64", false) {
		t.keep = true
		return
	}
	if enabled("ssa") && t.Ssadump() {
		t.keep = true
		return
	}
	if enabled("ssa") && enabled("exec") && t.SsadumpExec() {
		t.keep = true
		return
	}
	if enabled("gofmt") && t.Gofmt() {
		t.keep = true
		return
	}
}

func enabled(what string) bool {
	return *checkers == "all" || strings.Contains(*checkers, what)
}

func (t *Test) generateSource() bool {
	args := []string{"-seed", t.seed, "-dir", t.path}
	out, err := exec.Command("gosmith", args...).CombinedOutput()
	if err != nil {
		log.Printf("failed to execute gosmith for seed %v: %v\n%v\n", t.seed, err, string(out))
		return false
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Printf("failed to pwd: %v", err)
		return false
	}
	t.gopath = filepath.Join(pwd, t.path)
	return true
}

func (t *Test) Build(compiler, goos, goarch string, race bool) bool {
	typ := compiler + "." + goos + "." + goarch
	if race {
		typ += ".race"
	}
	outbin := filepath.Join(t.path, "bin"+typ)
	args := []string{"build", "-o", outbin, "-compiler", compiler}
	if race {
		args = append(args, "-race")
	}
	args = append(args, "main")
	cmd := exec.Command("go", args...)
	cmd.Env = []string{"GOARCH=" + goarch, "GOPATH=" + t.gopath + ":" + os.Getenv("GOPATH")}
	if goos != "" {
		cmd.Env = append(cmd.Env, "GOOS="+goos)
	}
	cmd.Env = append(cmd.Env, os.Environ()...)
	out, err := runWithTimeout(cmd)
	if err == nil {
		return false
	}
	for _, known := range knownBuildBugs[typ] {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	for _, known := range knownBuildBugs[compiler] {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	for _, known := range knownBuildBugs["all"] {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	outf, err := os.Create(filepath.Join(t.path, typ))
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

func (t *Test) Exec(compiler, goos, goarch string, race bool) bool {
	typ := compiler + "." + goos + "." + goarch
	if race {
		typ += ".race"
	}
	outbin := filepath.Join(t.path, "bin"+typ)
	if _, err := os.Stat(outbin); err != nil {
		return false
	}
	cmd := exec.Command(outbin)
	if goos == "nacl" {
		cmd = exec.Command("bash", "go_nacl_"+goarch+"_exec", outbin)
	}
	cmd.Env = []string{"GOMAXPROCS=2", "GOGC=0"}
	cmd.Env = append(cmd.Env, os.Environ()...)
	out, err := runWithTimeout(cmd)
	if err == nil {
		return false
	}
	for _, known := range knownExecBugs {
		if known.Match(out) {
			return false
		}
	}
	outf, err := os.Create(filepath.Join(t.path, "exec."+typ))
	if err != nil {
		log.Printf("failed to create output file: %v", err)
	} else {
		outf.Write(out)
		outf.Close()
	}
	log.Printf("%v exec failed, seed %v\n", typ, t.seed)
	atomic.AddUint64(&statExec, 1)
	return true
}

func (t *Test) Ssadump() bool {
	cmd := exec.Command("ssadump", "-build=CDPF", "main")
	cmd.Env = []string{"GOPATH=" + t.gopath}
	cmd.Env = append(cmd.Env, os.Environ()...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return false
	}
	for _, known := range knownSsadumpBugs {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	outf, err := os.Create(filepath.Join(t.path, "ssadump"))
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

func (t *Test) SsadumpExec() bool {
	cmd := exec.Command("ssadump", "-run", "main")
	cmd.Env = []string{"GOPATH=" + t.gopath, "GOMAXPROCS=2", "GOGC=10"}
	cmd.Env = append(cmd.Env, os.Environ()...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return false
	}
	for _, known := range knownExecBugs {
		if known.Match(out) {
			atomic.AddUint64(&statKnown, 1)
			return false
		}
	}
	outf, err := os.Create(filepath.Join(t.path, "ssadump.run"))
	if err != nil {
		log.Printf("failed to create output file: %v", err)
	} else {
		outf.Write(out)
		outf.Close()
	}
	log.Printf("ssadump.run failed, seed %v\n", t.seed)
	atomic.AddUint64(&statSsadump, 1)
	return true
}

func (t *Test) Gofmt() bool {
	files := []string{"main/0.go" /*, "main/1.go", "main/2.go", "a/0.go", "a/1.go", "a/2.go", "b/0.go", "b/1.go", "b/2.go"*/}
	for _, f := range files {
		if t.GofmtFile(filepath.Join(t.path, "src", f)) {
			return true
		}
	}
	return false
}

func (t *Test) GofmtFile(fname string) bool {
	formatted, err := exec.Command("gofmt", fname).CombinedOutput()
	if err != nil {
		outf, err := os.Create(fname + ".gofmt")
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
	fname1 := fname + ".formatted"
	outf, err := os.Create(fname1)
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return false
	}
	outf.Write(formatted)
	outf.Close()

	formatted2, err := exec.Command("gofmt", fname1).CombinedOutput()
	if err != nil {
		outf, err := os.Create(fname + ".gofmt")
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
	outf2, err := os.Create(fname + ".formatted2")
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return false
	}
	outf2.Write(formatted2)
	outf2.Close()

	// Fails too often due to https://code.google.com/p/go/issues/detail?id=8021
	if true {
		if bytes.Compare(formatted, formatted2) != 0 {
			log.Printf("nonidempotent gofmt, seed %v\n", t.seed)
			atomic.AddUint64(&statGofmt, 1)
			return true
		}
	}

	removeWs := func(r rune) rune {
		// Chars that gofmt can remove/shuffle.
		if r == ' ' || r == '\t' || r == '\n' || r == '(' || r == ')' || r == ',' || r == ';' {
			return -1
		}
		return r
	}
	origfile, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Printf("failed to read file: %v", err)
	}
	stripped := bytes.Map(removeWs, origfile)
	stripped2 := bytes.Map(removeWs, formatted)
	if bytes.Compare(stripped, stripped2) != 0 {
		writeStrippedFile(fname+".stripped0", stripped)
		writeStrippedFile(fname+".stripped1", stripped2)
		log.Printf("corrupting gofmt, seed %v\n", t.seed)
		atomic.AddUint64(&statGofmt, 1)
		return true
	}
	return false
}

func writeStrippedFile(fn string, data []byte) {
	f, err := os.Create(fn)
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return
	}
	defer f.Close()
	const lineSize = 80
	for i := 0; i < len(data); i += lineSize {
		end := i + lineSize
		if end > len(data) {
			end = len(data)
		}
		f.Write(data[i:end])
		f.Write([]byte{'\n'})
	}
}

func runWithTimeout(cmd *exec.Cmd) ([]byte, error) {
	var bufout bytes.Buffer
	var buferr bytes.Buffer
	cmd.Stdout = &bufout
	cmd.Stderr = &buferr
	cmd.Start()
	done := make(chan bool)
	defer close(done)
	go func() {
		select {
		case <-done:
			return
		case <-time.After(time.Duration(*timeout) * time.Second):
		}
		cmd.Process.Signal(syscall.SIGABRT)
		select {
		case <-done:
			return
		case <-time.After(5 * time.Second):
		}
		cmd.Process.Signal(syscall.SIGTERM)
	}()
	err := cmd.Wait()
	return []byte(bufout.String() + buferr.String()), err
}
