package main

import (
	"fmt"
	"math/rand"
)

func (c *Context) statement() {
	if c.stmtCount >= 30 {
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

func (c *Context) initStatements() {
	c.statements = []func() bool{
		c.stmtOas,
		c.stmtAs,
		c.stmtInc,
		c.stmtIf,
		c.stmtFor,
		c.stmtSend,
		c.stmtRecv,
		c.stmtSelect,
		c.stmtTypeDecl,
		c.stmtCall,
		c.stmtReturn,
	}
}

func (c *Context) stmtOas() bool {
	id := c.newId()
	if id == "_" {
		return false
	}
	typ := c.existingType()
	switch rand.Intn(2) {
	case 0: // short form
		c.F("%v := ", id)
	case 1: // full form
		c.F("var %v %v = ", id, typ.id)
	}
	c.expression(typ)
	c.F("\n")
	if id != "_" {
		c.vars = append(c.vars, &Var{id: id, typ: typ})
	}
	return true
}

func (c *Context) stmtAs() bool {
	types := c.aTypeList(TraitAny)
	c.F("%v = %v\n", c.formatLvalueList(types), c.formatRvalueList(types))
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

func (c *Context) stmtSend() bool {
	v, ok := c.existingVarClass(ClassChan)
	if !ok {
		return false
	}
	c.F("%v <- ", v.id)
	c.expression(v.typ.ktyp)
	c.F("\n")
	return true
}

func (c *Context) stmtRecv() bool {
	t := c.aType(TraitReceivable)
	c.F("%v", c.lvalue(t.ktyp))
	if c.rand(2) == 0 {
		c.F(", %v", c.lvalue(c.boolType))
	}
	c.F(" = <-%v\n", c.rvalue(t))
	return true
}

func (c *Context) stmtSelect() bool {
	c.F("select {\n")
	for rand.Intn(5) != 0 {
		if rand.Intn(2) == 0 {
			cv, ok := c.existingVarClass(ClassChan)
			if ok {
				c.F("case %v <- ", cv.id)
				c.expression(cv.typ.ktyp)
				c.F(":\n")
				c.block()
			}
		} else {
			cv, ok := c.existingVarClass(ClassChan)
			if ok {
				c.F("case ")
				vv, ok := c.existingVarType(cv.typ.ktyp)
				if ok {
					c.F("%v ", vv.id)
				} else {
					c.F("_")
				}
				if rand.Intn(2) == 0 {
					bv, ok := c.existingVarType(c.boolType)
					if ok {
						c.F(", %v ", bv.id)
					} else {
						c.F(", _")
					}
				}
				c.F(" = <-%v:\n", cv.id)
				c.block()
			}
		}
	}
	if rand.Intn(2) == 0 {
		c.F("default:\n")
		c.block()
	}
	c.F("}\n")
	return true
}

func (c *Context) stmtTypeDecl() bool {
	id := c.newId()
	t := c.aType(TraitAny)
	c.F("type %v %v\n", id, t.id)

	newTyp := new(Type)
	*newTyp = *t
	newTyp.id = id
	newTyp.literal = func() string {
		return fmt.Sprintf("%v(%v)", id, t.literal())
	}
	if id != "_" {
		c.types = append(c.types, newTyp)
	}
	return true
}

func (c *Context) stmtCall() bool {
	if c.rand(2) == 0 {
		return c.stmtCallBuiltin()
	}
	t := c.aType(TraitFunction)
	c.F("%v(%v)\n", c.rvalue(t), c.formatRvalueList(t.atyp))
	return true
}

func (c *Context) stmtCallBuiltin() bool {
	builtins := []string{"close", "copy", "delete", "panic", "print", "println", "recover"}
	switch fn := builtins[c.rand(len(builtins))]; fn {
	case "close":
		return false
	case "copy":
		return false
	case "delete":
		return false
	case "panic":
		return false
	case "print":
		fallthrough
	case "println":
		list := c.aTypeList(TraitPrintable)
		c.F("%v(%v)\n", fn, c.formatRvalueList(list))
		return false
	case "recover":
		return false
	default:
		panic("bad")
	}
}

func (c *Context) stmtReturn() bool {
	c.F("return ")
	for i, t := range c.retType {
		if i != 0 {
			c.F(",")
		}
		c.F("%v", c.rvalue(t))
	}
	c.F("\n")
	return true
}
