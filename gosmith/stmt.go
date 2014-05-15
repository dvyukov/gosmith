package main

import (
	"fmt"
	"math/rand"
)

func (c *Context) statement() {
	if c.stmtCount >= 20 {
		return
	}
	c.stmtCount++
	for {
		f := c.statements[rand.Intn(len(c.statements))]
		if !f() {
			continue
		}
		break
	}
}

func (c *Context) collectStatements() {
	c.statements = []func() bool{
		c.stmtOas,
		c.stmtAs,
		c.stmtInc,
		c.stmtIf,
		c.stmtFor,
		c.stmtTypeDecl,
	}
}

func (c *Context) stmtOas() bool {
	id := c.newId()
	typ := c.existingType()
	switch rand.Intn(2) {
	case 0: // short form
		c.F("%v := ", id)
	case 1: // full form
		c.F("var %v %v = ", id, typ.id)
	}
	c.expression(typ)
	c.F("\n")
	c.vars = append(c.vars, &Var{id: id, typ: typ})
	return true
}

func (c *Context) stmtAs() bool {
	v, ok := c.existingVar()
	if !ok {
		return false
	}
	c.F("%v = ", v.id)
	c.expression(v.typ)
	c.F("\n")
	return true
}

func (c *Context) stmtInc() bool {
	v, ok := c.existingVarClass(ClassNumeric)
	if !ok {
		return false
	}
	c.F("%v", v.id)
	switch rand.Intn(2) {
	case 0:
		c.F("++\n")
	case 1:
		c.F("--\n")
	}
	return true
}

func (c *Context) stmtIf() bool {
	c.F("if ")
	bt, _ := c.existingTypeClass(ClassBoolean)
	c.expression(bt)
	c.F(" {\n")
	c.block()
	c.F("}\n")
	return true
}

func (c *Context) stmtFor() bool {
	c.F("for ")
	bt, _ := c.existingTypeClass(ClassBoolean)
	c.expression(bt)
	c.F(" {\n")
	c.block()
	c.F("}\n")
	return true
}

func (c *Context) stmtTypeDecl() bool {
	id := c.newId()
	c.F("type %v ", id)
	switch rand.Intn(2) {
	case 0: // alias
		typ := c.existingType()
		newTyp := new(Type)
		*newTyp = *typ
		newTyp.id = id
		newTyp.literal = func() string {
			return fmt.Sprintf("%v(%v)", id, typ.literal())
		}
		c.types = append(c.types, newTyp)
		c.F("%v", typ.id)
	case 1: // map
		ktyp, _ := c.existingTypeClass(ClassNumeric)
		vtyp := c.existingType()
		typ := &Type{id: Id(fmt.Sprintf("map[%v]%v", ktyp.id, vtyp.id)), class: ClassMap, literal: func() string {
			return fmt.Sprintf("map[%v]%v{}", ktyp.id, vtyp.id)
		}}
		c.types = append(c.types, typ)
		c.F("%v", typ.id)
	}
	c.F("\n")
	return true
}
