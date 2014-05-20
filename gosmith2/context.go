package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const (
	NPackages       = 1
	NFiles          = 1
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
	str           string
	parent        *Block
	extendable    bool
	isBreakable   bool
	isContinuable bool
	sub           []*Block
	consts        []*Const
	types         []*Type
	funcs         []*Func
	vars          []*Var
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
	curPackage  int
	curBlock    *Block
	curBlockPos int
	curFunc     *Func

	packages    [NPackages]*Package
	toplevVars  []*Var
	toplevFuncs []*Func

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
	efaceType       *Type
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
	//packages[1] = newPackage("a")
	//packages[2] = newPackage("b")
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
	b := &Block{parent: curBlock, str: s}
	if curBlockPos+1 == len(curBlock.sub) {
		curBlock.sub = append(curBlock.sub, b)
	} else {
		curBlock.sub = append(curBlock.sub, nil)
		copy(curBlock.sub[curBlockPos+2:], curBlock.sub[curBlockPos+1:])
		curBlock.sub[curBlockPos+1] = b
	}
	curBlockPos++
}

func resetContext(pi int) {
	curPackage = pi
	p := packages[pi]
	curBlock = p.top
	curBlockPos = len(curBlock.sub) - 1
	curFunc = nil
}

func genToplevFunction(pi int, f *Func) {
	resetContext(pi)
	curFunc = f
	enterBlock(true)
	enterBlock(true)
	argIds := make([]string, len(f.args))
	argStr := ""
	for i, a := range f.args {
		argIds[i] = newId()
		if i != 0 {
			argStr += ", "
		}
		argStr += argIds[i] + " " + a.id
	}	
	line("func %v(%v)%v {", f.name, argStr, fmtTypeList(f.rets, false))
	for i, a := range f.args {
		defineVar(argIds[i], a)
	}
	genBlock()
	leaveBlock()
	stmtReturn()
	line("}")
	leaveBlock()
}

func genToplevVar(pi int, v *Var) {
	panic("bad")
	resetContext(pi)
	enterBlock(true)
	line("var %v = %v", v.id, rvalue(v.typ))
	toplevVars = append(toplevVars, v)
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
				fmt.Fprintf(w, "var SINK interface{}\n")
			}
		}
		for _, decl := range p.top.sub {
			serializeBlock(files[rnd(len(files))], decl, 0)
		}
	}
}

func serializeBlock(w *bufio.Writer, b *Block, d int) {
	if true {
		if b.str != "" {
			w.WriteString(b.str)
			w.WriteString("\n")
		}
	} else {
		w.WriteString("/*" + strings.Repeat("*", d) + "*/ ")
		w.WriteString(b.str)
		w.WriteString(F(" // ext=%v vars=%v types=%v", b.extendable, len(b.vars), len(b.types)))
		w.WriteString("\n")
	}
	for _, b1 := range b.sub {
		serializeBlock(w, b1, d+1)
	}
}

func vars() []*Var {
	var vars []*Var
	vars = append(vars, toplevVars...)
	var f func(b *Block, pos int)
	f = func(b *Block, pos int) {
		for _, b1 := range b.sub[:pos+1] {
			vars = append(vars, b1.vars...)
		}
		if b.parent != nil {
			f(b.parent, len(b.parent.sub)-1)
		}
	}
	f(curBlock, curBlockPos)
	return vars
}

func types() []*Type {
	var types []*Type
	types = append(types, predefinedTypes...)
	var f func(b *Block, pos int)
	f = func(b *Block, pos int) {
		for _, b1 := range b.sub[:pos+1] {
			types = append(types, b1.types...)
		}
		if b.parent != nil {
			f(b.parent, len(b.parent.sub)-1)
		}
	}
	f(curBlock, curBlockPos)
	return types
}

func defineVar(id string, t *Type) {
	v := &Var{id: id, typ: t}
	b := curBlock.sub[curBlockPos]
	b.vars = append(b.vars, v)
	//vars = append(vars, v)
}

func defineType(t *Type) {
	b := curBlock.sub[curBlockPos]
	b.types = append(b.types, t)
	//types = append(types, t)
}

func materializeVar(t *Type) string {
	// TODO: reset exprDepth and friends
	// TODO: generate var in another package
	id := newId()
	if true {
		curBlock0 := curBlock
		curBlockPos0 := curBlockPos
		curBlockLen0 := len(curBlock.sub)
		defer func() {
			if curBlock == curBlock0 {
				curBlockPos0 += len(curBlock.sub) - curBlockLen0
			}
			curBlock = curBlock0
			curBlockPos = curBlockPos0
		}()
	loop:
		for {
			if curBlock.parent == nil {
				break
			}
			if !curBlock.extendable || curBlockPos < 0 {
				curBlock = curBlock.parent
				curBlockPos = len(curBlock.sub) - 2
				continue
			}
			if rnd(3) == 0 {
				break
			}
			if curBlockPos >= 0 {
				b := curBlock.sub[curBlockPos]
				for _, t1 := range b.types {
					if dependsOn(t, t1) {
						break loop
					}
				}
			}
			curBlockPos--
		}
		if curBlock.parent == nil {
			enterBlock(true)
			line("var %v = %v", id, rvalue(t))
			toplevVars = append(toplevVars, &Var{id: id, typ: t})
			leaveBlock()
			//packages[curPackage].undefVars = append(packages[curPackage].undefVars, &Var{id: id, typ: t})
		} else {
			line("%v := %v", id, rvalue(t))
			defineVar(id, t)
		}
		return id
	}

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

func materializeFunc(res *Type) *Func {
	f := &Func{name: newId(), args: atypeList(TraitGlobal), rets: []*Type{res}}

	curBlock0 := curBlock
	curBlockPos0 := curBlockPos
	curFunc0 := curFunc
	defer func() {
		curBlock = curBlock0
		curBlockPos = curBlockPos0
		curFunc = curFunc0
	}()
	genToplevFunction(curPackage, f)
	return f
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
	b := &Block{parent: curBlock, extendable: !nonextendable}
	b.isBreakable = curBlock.isBreakable
	b.isContinuable = curBlock.isContinuable
	curBlock.sub = append(curBlock.sub, b)
	curBlock = b
	curBlockPos = -1
}

func leaveBlock() {
	for _, b := range curBlock.sub {
		for _, v := range b.vars {
			line("_ = %v", v.id)
		}
	}

	curBlock = curBlock.parent
	curBlockPos = len(curBlock.sub) - 1
}
