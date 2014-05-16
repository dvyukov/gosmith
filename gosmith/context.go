package main

import (
	"fmt"
	"io"
	"math/rand"
)

type Id string

type Var struct {
	id   Id
	typ  *Type
	used bool
}

type Context struct {
	w              io.Writer
	incorrect      bool
	nonterminating bool
	idSeq          int
	typeDepth      int
	exprDepth      int
	stmtCount      int
	retType        []*Type
	boolType       *Type
	intType        *Type

	statements  []func() bool
	expressions []func(res *Type) bool

	vars       []*Var
	varScopes  []int
	types      []*Type
	typeScopes []int
}

func NewContext(w io.Writer, incorrect, nonterminating bool) *Context {
	c := &Context{w: w, incorrect: incorrect, nonterminating: nonterminating}
	c.initTypes()
	c.initExpressions()
	c.initStatements()
	return c
}

func (c *Context) F(f string, args ...interface{}) {
	fmt.Fprintf(c.w, f, args...)
}

func (c *Context) rand(n int) int {
	return rand.Intn(n)
}

func (c *Context) program() {
	c.F("package main\n")
	c.F("import \"unsafe\"\n")
	c.F("type uptr unsafe.Pointer\n")
	for rand.Intn(5) != 0 {
		c.stmtTypeDecl()
	}
	c.function("main", nil)
	c.function("foo", []*Type{c.aType(TraitAny)})
}

func (c *Context) function(name Id, ret []*Type) {
	c.retType = ret
	c.stmtCount = 0
	c.F("func %v() (", name)
	for i, t := range ret {
		if i != 0 {
			c.F(",")
		}
		c.F("%v", t.id)
	}
	c.F(") {\n")
	c.block()
	c.stmtReturn()
	c.F("}\n\n")
}

func (c *Context) block() {
	c.EnterScope()
	for rand.Intn(10) != 0 {
		c.statement()
	}
	c.LeaveScope()
}

func (c *Context) EnterScope() {
	c.varScopes = append(c.varScopes, len(c.vars))
	c.typeScopes = append(c.typeScopes, len(c.types))
}

func (c *Context) LeaveScope() {
	varLast := len(c.varScopes) - 1
	varIdx := c.varScopes[varLast]
	for _, v := range c.vars[varIdx:] {
		if !v.used {
			c.F("_ = %v\n", v.id)
		}

	}
	c.vars = c.vars[:c.varScopes[varLast]]
	c.varScopes = c.varScopes[:varLast]

	typeLast := len(c.typeScopes) - 1
	c.types = c.types[:c.typeScopes[typeLast]]
	c.typeScopes = c.typeScopes[:typeLast]
}

func (c *Context) newId() Id {
	if rand.Intn(3) == 0 {
		return "_"
	}
	c.idSeq++
	return Id(fmt.Sprintf("id%v", c.idSeq))
}

func (c *Context) existingType() *Type {
	return c.types[rand.Intn(len(c.types))]
}

func (c *Context) existingTypeComparable() *Type {
	for _, t := range c.types {
		if t.class == ClassBoolean || t.class == ClassNumeric || t.class == ClassString ||
			t.class == ClassPointer || t.class == ClassChan || t.class == ClassInterface {
			return t
		}
	}
	return nil
}

func (c *Context) existingTypeOrdered() *Type {
	for _, t := range c.types {
		if t.class == ClassNumeric || t.class == ClassString {
			return t
		}
	}
	return nil
}

func (c *Context) existingTypeClass(cl TypeClass) (*Type, bool) {
	for _, t := range c.types {
		if t.class == cl {
			return t, true
		}
	}
	return nil, false
}

func (c *Context) existingVarType(typ *Type) (*Var, bool) {
	for _, v := range c.vars {
		if v.typ == typ {
			return v, true
		}
	}
	return nil, false
}

func (c *Context) existingVarClass(cl TypeClass) (*Var, bool) {
	for _, v := range c.vars {
		if v.typ.class == cl {
			return v, true
		}
	}
	return nil, false
}
