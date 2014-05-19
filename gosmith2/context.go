package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

const (
	NPackages    = 3
	NFiles       = 3
	NStatements  = 30
	NExpressions = 8
)

type Package struct {
	name    string
	imports map[string]bool
	top     *Block

	undefFuncs []*Func
	undefVars  []*Var
}

type Block struct {
	str    string
	parent *Block
	sub    []*Block
	consts []*Const
	types  []*Type
	funcs  []*Func
	vars   []*Var
}

type Func struct {
	name string
	args []*Type
	rets []*Type
}

type Var struct {
	id    string
	typ   *Type
	block *Block
}

type Const struct {
}

var (
	curPackage int
	curBlock   *Block
	curFunc    *Func

	packages    [NPackages]*Package
	consts      []*Const
	constScopes []int
	types       []*Type
	typeScopes  []int
	vars        []*Var
	varScopes   []int

	idSeq       int
	typeDepth   int
	stmtCount   int
	exprDepth   int
	boolType    *Type
	intType     *Type
	statements  []func()
	expressions []func(res *Type) string
)

func writeProgram(dir string) {
	initTypes()
	initExpressions()
	initStatements()
	initProgram()
	for pi := range packages {
		genPackage(pi)
	}
	serializeProgram(dir)
}

func initProgram() {
	packages[0] = newPackage("main")
	packages[0].undefFuncs = []*Func{&Func{name: "main", args: []*Type{}, rets: []*Type{}}}
	packages[1] = newPackage("a")
	packages[2] = newPackage("b")
}

func newPackage(name string) *Package {
	return &Package{name: name, imports: make(map[string]bool), top: &Block{}}
}

func genPackage(pi int) {
	p := packages[pi]
	for len(p.undefFuncs) != 0 || len(p.undefVars) != 0 {
		if len(p.undefFuncs) != 0 {
			f := p.undefFuncs[len(p.undefFuncs)-1]
			p.undefFuncs = p.undefFuncs[:len(p.undefFuncs)-1]
			genToplevFunction(pi, f)
		}
		if len(p.undefVars) != 0 {
			v := p.undefVars[len(p.undefVars)-1]
			p.undefVars = p.undefVars[:len(p.undefVars)-1]
			genToplevVar(pi, v)
		}
	}
}

func line(f string, args ...interface{}) *Block {
	s := fmt.Sprintf(f, args...)
	b := &Block{str: s}
	curBlock.sub = append(curBlock.sub, b)
	return b
}

func resetContext(pi int) {
	curPackage = pi
	p := packages[pi]
	curBlock = p.top
	curFunc = nil
}

func genToplevFunction(pi int, f *Func) {
	resetContext(pi)
	curFunc = f
	enterBlock(true)
	line("func %v() %v {", f.name, fmtTypeList(f.rets, false))
	genBlock()
	stmtReturn()
	line("}")
	leaveBlock()
}

func genToplevVar(pi int, v *Var) {
	resetContext(pi)
	enterBlock(true)
	// TODO: add all other toplev vars and funcs to context
	line("var %v = %v", v.id, rvalue(v.typ))
	leaveBlock()
}

func genBlock() {
	enterBlock(false)
	for rnd(10) != 0 {
		genStatement()
	}
	leaveBlock()
}

func serializeProgram(dir string) {
	for _, p := range packages {
		path := dir
		if p.name != "main" {
			path = filepath.Join(dir, "src", p.name)
		}
		os.MkdirAll(path, os.ModePerm)
		files := [NFiles]*bufio.Writer{}
		for i := range files {
			fname := filepath.Join(path, fmt.Sprintf("%v.go", i))
			f, err := os.Create(fname)
			if err != nil {
				fmt.Fprintf(os.Stdout, "failed to create a file: %v\n", err)
				os.Exit(1)
			}
			w := bufio.NewWriter(bufio.NewWriter(f))
			files[i] = w
			defer func() {
				w.Flush()
				f.Close()
			}()
			fmt.Fprintf(w, "package %s\n", p.name)
			for imp := range p.imports {
				fmt.Fprintf(w, "import \"%s\"\n", imp)
			}
			for imp := range p.imports {
				fmt.Fprintf(w, "var _ = %s.UsePackage\n", imp)
			}
			if i == 0 {
				fmt.Fprintf(w, "var UsePackage = 0\n")
			}
		}
		for _, decl := range p.top.sub {
			serializeBlock(files[rnd(len(files))], decl)
		}
	}
}

func serializeBlock(w *bufio.Writer, b *Block) {
	if b.str != "" {
		w.WriteString(b.str)
		w.WriteString("\n")
	}
	for _, b1 := range b.sub {
		serializeBlock(w, b1)
	}
}

func defineVar(t *Type) string {
	// TODO: generate var in another package
	id := newId()
	/*
	  if rndBool() {
	    for i := curPackage; i < NPackages; i++ {
	      if rndBool() || i == NPackages - 1 {
	        if i != curPackage {
	          packages[curPackage].imports[packages[i].name] = true
	          id = packages[i].name + ".I" + id
	        }
	        packages[i].undefVars = append(packages[i].undefVars, &Var{id: id, typ: t})
	        break
	      }
	    }
	  } else*/{
		if curBlock.parent.parent == nil {
			packages[curPackage].undefVars = append(packages[curPackage].undefVars, &Var{id: id, typ: t})
		} else {
			//curBlock0 := curBlock
			line("%v := %v", id, rvalue(t))
			vars = append(vars, &Var{id: id, typ: t})
			//curBlock = curBlock0
		}
	}
	return id
}

func rnd(n int) int {
	return rand.Intn(n)
}

func rndBool() bool {
	return rnd(2) == 0
}

func choice(ch ...string) string {
	return ch[rnd(len(ch))]
}

func newId() string {
	idSeq++
	return fmt.Sprintf("id%v", idSeq)
}

func enterBlock(stickyFirstLine bool) {
	varScopes = append(varScopes, len(vars))
	typeScopes = append(typeScopes, len(types))

	b := &Block{parent: curBlock}
	curBlock.sub = append(curBlock.sub, b)
	curBlock = b
	if !stickyFirstLine {
		line("")
	}
}

func leaveBlock() {
	varLast := len(varScopes) - 1
	varIdx := varScopes[varLast]
	for _, v := range vars[varIdx:] {
		line("_ = %v", v.id)
	}
	vars = vars[:varScopes[varLast]]
	varScopes = varScopes[:varLast]

	typeLast := len(typeScopes) - 1
	types = types[:typeScopes[typeLast]]
	typeScopes = typeScopes[:typeLast]

	curBlock = curBlock.parent
}
