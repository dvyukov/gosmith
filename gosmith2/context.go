package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

const (
	NPackages       = 3
	NFiles          = 3
	NStatements     = 30
	NExprDepth      = 8
	NExprCount      = 20
	NTotalExprCount = NStatements * NExprCount
)

type Package struct {
	name    string
	imports map[string]bool
	top     *Block

	undefFuncs []*Func
	undefVars  []*Var
}

type Block struct {
	str        string
	parent     *Block
	extendable bool
	sub        []*Block
	consts     []*Const
	types      []*Type
	funcs      []*Func
	vars       []*Var
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

	packages [NPackages]*Package
	/*
		consts      []*Const
		constScopes []int
		types       []*Type
		typeScopes  []int
		//vars        []*Var
		//varScopes   []int
	*/

	idSeq           int
	typeDepth       int
	stmtCount       int
	exprDepth       int
	exprCount       int
	totalExprCount  int
	predefinedTypes []*Type
	stringType      *Type
	boolType        *Type
	intType         *Type
	byteType        *Type
	statements      []func()
	expressions     []func(res *Type) string
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
	return &Package{name: name, imports: make(map[string]bool), top: &Block{extendable: true}}
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

func F(f string, args ...interface{}) string {
	return fmt.Sprintf(f, args...)
}

func line(f string, args ...interface{}) {
	s := F(f, args...)
	b := &Block{str: s}
	curBlock.sub = append(curBlock.sub, b)
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
		path := filepath.Join(dir, "src", p.name)
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
		//w.WriteString(F(" // vars=%+v types=%+v", b.vars, b.types))
		w.WriteString("\n")
	}
	for _, b1 := range b.sub {
		serializeBlock(w, b1)
	}
}

func vars() []*Var {
	var vars []*Var
	var f func(b *Block)
	f = func(b *Block) {
		for _, b1 := range b.sub {
			vars = append(vars, b1.vars...)
		}
	}
	f(curBlock)
	return vars
}

func types() []*Type {
	var types []*Type
	types = append(types, predefinedTypes...)
	var f func(b *Block)
	f = func(b *Block) {
		for _, b1 := range b.sub {
			types = append(types, b1.types...)
		}
	}
	f(curBlock)
	return types
}

func defineVar(id string, t *Type) {
	v := &Var{id: id, typ: t}
	b := curBlock.sub[len(curBlock.sub)-1]
	b.vars = append(b.vars, v)
	//vars = append(vars, v)
}

func defineType(t *Type) {
	b := curBlock.sub[len(curBlock.sub)-1]
	b.types = append(b.types, t)
	//types = append(types, t)
}

func materializeVar(t *Type) string {
	// TODO: reset exprDepth and friends
	// TODO: generate var in another package
	id := newId()
	/*
		curBlock0 := curBlock
		patching0 := patching
		patching = true
		defer func() {
			curBlock = curBlock0
			patching = patching0
		}()
		for {
			if !curBlock.extendable {
				curBlock = curBlock.parent
				continue
			}
			if rndBool() {
				break
			}
		}
	*/

	// TODO: this code does not respect type scope,
	// e.g. it tries to emit a var into another package when the type is function-local
	/*if rndBool() {
		for i := curPackage; i < NPackages; i++ {
			if rndBool() || i == NPackages-1 {
				id = "I" + id
				packages[i].undefVars = append(packages[i].undefVars, &Var{id: id, typ: t})
				if i != curPackage {
					packages[curPackage].imports[packages[i].name] = true
					id = packages[i].name + "." + id
				}
				break
			}
		}
	} else*/{
		if curBlock.parent.parent == nil || len(curBlock.sub) == 0 {
			packages[curPackage].undefVars = append(packages[curPackage].undefVars, &Var{id: id, typ: t})
		} else {
			line("%v := %v", id, rvalue(t))
			defineVar(id, t)
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

func enterBlock(nonextendable bool) {
	/*
		varScopes = append(varScopes, len(vars))
		typeScopes = append(typeScopes, len(types))
	*/

	b := &Block{parent: curBlock, extendable: !nonextendable}
	curBlock.sub = append(curBlock.sub, b)
	curBlock = b
}

func leaveBlock() {
	/*
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
	*/

	for _, b := range curBlock.sub {
		for _, v := range b.vars {
			line("_ = %v", v.id)
		}
	}

	curBlock = curBlock.parent
}