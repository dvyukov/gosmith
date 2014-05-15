package main

import (
	"math/rand"
)

func (c *Context) expression(res *Type) {
	if c.exprDepth >= 10 {
		c.F("%v", res.literal())
		return
	}
	c.exprDepth++
	for {
		f := c.expressions[rand.Intn(len(c.expressions))]
		if !f(res) {
			continue
		}
		break
	}
	c.exprDepth--
}

func (c *Context) collectExpressions() {
	c.expressions = []func(res *Type) bool{
		c.exprLiteral,
		c.exprVar,
		c.exprArith,
		c.exprEqual,
		c.exprOrder,
	}
}

func (c *Context) exprLiteral(res *Type) bool {
	c.F("%v", res.literal())
	return true
}

func (c *Context) exprVar(res *Type) bool {
	v, ok := c.existingVarType(res)
	if !ok {
		return false
	}
	c.F("%v", v.id)
	return true
}

func (c *Context) exprArith(res *Type) bool {
	if res.class != ClassNumeric {
		return false
	}
	c.F("((")
	c.expression(res)
	c.F(")")
	switch rand.Intn(3) {
	case 0:
		c.F(" + ")
	case 1:
		c.F(" + ")
	case 2:
		c.F(" * ")
	}
	c.F("(")
	c.expression(res)
	c.F("))")
	return true
}

func (c *Context) exprEqual(res *Type) bool {
	if res != c.boolType {
		return false
	}
	typ := c.existingTypeComparable()
	c.F("((")
	c.expression(typ)
	c.F(")")
	switch rand.Intn(2) {
	case 0:
		c.F(" == ")
	case 1:
		c.F(" != ")
	}
	c.F("(")
	c.expression(typ)
	c.F("))")
	return true
}

func (c *Context) exprOrder(res *Type) bool {
	if res != c.boolType {
		return false
	}
	typ := c.existingTypeOrdered()
	c.F("((")
	c.expression(typ)
	c.F(")")
	switch rand.Intn(4) {
	case 0:
		c.F(" < ")
	case 1:
		c.F(" <= ")
	case 2:
		c.F(" > ")
	case 3:
		c.F(" >= ")
	}
	c.F("(")
	c.expression(typ)
	c.F("))")
	return true
}
