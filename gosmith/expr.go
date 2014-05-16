package main

import (
	"bytes"
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

func (c *Context) initExpressions() {
	c.expressions = []func(res *Type) bool{
		c.exprLiteral,
		c.exprRecv,
		c.exprVar,
		c.exprArith,
		c.exprEqual,
		c.exprOrder,
		c.exprCall,
		c.exprIndex,
	}
}

func (c *Context) rvalue(t *Type) string {
	var buf bytes.Buffer
	w := c.w
	c.w = &buf
	c.expression(t)
	c.w = w
	return buf.String()
}

func (c *Context) lvalue(t *Type) string {
	v, ok := c.existingVarType(t)
	if !ok {
		return "_"
	}
	return string(v.id)
}

func (c *Context) exprLiteral(res *Type) bool {
	c.F("%v", res.literal())
	return true
}

func (c *Context) exprRecv(res *Type) bool {
	t := c.chanOf(res)
	c.F("<-%v", c.rvalue(t))
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

func (c *Context) exprCall(ret *Type) bool {
	if c.rand(2) == 0 {
		return c.exprCallBuiltin(ret)
	}
	return false
}

func (c *Context) exprCallBuiltin(ret *Type) bool {
	builtins := []string{"append", "cap", "complex", "copy",
		"imag", "len", "make", "new", "real", "recover"}
	switch fn := builtins[c.rand(len(builtins))]; fn {
	case "append":
		return false
	case "cap":
		fallthrough
	case "len":
		if ret != c.intType { // TODO: must be convertable
			return false
		}
		t := c.aType(TraitLenCapable)
		if (t.class == ClassString || t.class == ClassMap) && fn == "cap" {
			return false

		}
		c.F("%v(%v)", fn, c.rvalue(t))
		return true
	case "complex":
		return false
	case "copy":
		return false
	case "imag":
		return false
	case "make":
		return false
	case "new":
		return false
	case "real":
		return false
	case "recover":
		return false
	default:
		panic("bad")
	}
}

func (c *Context) exprIndex(ret *Type) bool {
	options := []string{"array", "slice", "ptr to array", "string", "map"}
	switch what := options[c.rand(len(options))]; what {
	case "array":
		return false
	case "slice":
		t := c.sliceOf(ret)
		c.F("(%v)[%v]", c.rvalue(t), c.rvalue(c.intType))
		return true
	case "ptr to array":
		return false
	case "string":
		return false
	case "map":
		return false
	default:
		panic("bad")
	}
}
