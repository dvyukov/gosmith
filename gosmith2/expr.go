package main

import (
	"bytes"
	"fmt"
)

func initExpressions() {
	expressions = []func(res *Type) string{
		exprLiteral,
		exprVar,
		//c.exprRecv,
		//c.exprVar,
		//c.exprArith,
		//c.exprEqual,
		//c.exprOrder,
		//c.exprCall,
		exprIndexSlice,
	}
}

func expression(res *Type) string {
	if exprDepth >= NExpressions {
		return exprLiteral(res)
	}
	exprDepth++
	s := expressions[rnd(len(expressions))](res)
	exprDepth--
	return s
}

func rvalue(t *Type) string {
	return expression(t)
}

func lvalue(t *Type) string {
	// TODO: check existing vars
	return defineVar(t)
}

func fmtRvalueList(list []*Type) string {
	var buf bytes.Buffer
	for i, t := range list {
		if i != 0 {
			buf.Write([]byte{','})
		}
		fmt.Fprintf(&buf, "%v", rvalue(t))
	}
	return buf.String()
}

func fmtLvalueList(list []*Type) string {
	var buf bytes.Buffer
	for i, t := range list {
		if i != 0 {
			buf.Write([]byte{','})
		}
		buf.WriteString(lvalue(t))
	}
	return buf.String()
}

func exprLiteral(res *Type) string {
	return res.literal()
}

func exprVar(res *Type) string {
	// TODO: check existing vars
	return defineVar(res)
}

/*
func (c *Context) exprRecv(res *Type) bool {
	t := c.chanOf(res)
	c.F("(<-%v)", c.rvalue(t))
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
	c.F("(")
	c.expression(res)
	switch rand.Intn(3) {
	case 0:
		c.F(" + ")
	case 1:
		c.F(" + ")
	case 2:
		c.F(" * ")
	}
	c.expression(res)
	c.F(")")
	return true
}

func (c *Context) exprEqual(res *Type) bool {
	if res != c.boolType {
		return false
	}
	typ := c.existingTypeComparable()
	c.F("(")
	c.expression(typ)
	switch rand.Intn(2) {
	case 0:
		c.F(" == ")
	case 1:
		c.F(" != ")
	}
	c.expression(typ)
	c.F(")")
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
*/

func exprIndexSlice(ret *Type) string {
	//options := []string{"array", "slice", "ptr to array", "string", "map"}

	t := sliceOf(ret)
	return fmt.Sprintf("(%v)[%v]", rvalue(t), rvalue(intType))
}
