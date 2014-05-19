package main

import ()

func initStatements() {
	statements = []func(){
		stmtOas,
		stmtAs,
		//stmtInc,
		//stmtIf,
		//stmtFor,
		//stmtSend,
		//c.stmtRecv,
		//c.stmtSelect,
		//c.stmtTypeDecl,
		//c.stmtCall,
		//c.stmtReturn,
	}
}

func genStatement() {
	if stmtCount >= NStatements {
		return
	}
	stmtCount++
	statements[rnd(len(statements))]()
}

func stmtOas() {
	id := newId()
	t := atype(TraitAny)
	b := line("%v := %v", id, rvalue(t))
	v := &Var{id: id, typ: t}
	b.vars = append(b.vars, v)
	vars = append(vars, v)
}

func stmtReturn() {
	line("return %v", fmtRvalueList(curFunc.rets))
}

func stmtAs() {
	types := atypeList(TraitAny)
	line("%v = %v", fmtLvalueList(types), fmtRvalueList(types))
}

/*
func (c *Context) stmtInc() bool {
  //b := newBlock("%v %v", lvalue(atype(ClassArith)), choice("--", "++"))

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
  //enterBlock(true)
  //line("if %v {\n", rvalue(atype(ClassBoolean)))
  //genBlock()
  //if randBool() {
  //  line("} else {")
  //  genBlock()
  //}
  //line("}")
  //leaveBlock()

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
	t := c.aType(TraitSendable)
	c.F("%v <- %v\n", c.rvalue(t), c.rvalue(t.ktyp))
	return true
}

func (c *Context) stmtRecv() bool {
	t := c.aType(TraitReceivable)
	ch := c.rvalue(t)
	switch c.choice("single", "double", "single_decl", "double_decl") {
	case "single":
		c.F("%v = <-%v\n", c.lvalue(t.ktyp), ch)
	case "double":
		c.F("%v, %v = <-%v\n", c.lvalue(t.ktyp), c.lvalue(c.boolType), ch)
	case "single_decl":
		vv := &Var{id: c.newId(), typ: t.ktyp}
		c.F("%v := <-%v\n", vv.id, ch)
		c.vars = append(c.vars, vv)
	case "double_decl":
		vv := &Var{id: c.newId(), typ: t.ktyp}
		ok := &Var{id: c.newId(), typ: c.boolType}
		c.F("%v, %v := <-%v\n", vv.id, ok.id, ch)
		c.vars = append(c.vars, vv)
		c.vars = append(c.vars, ok)
	default:
		panic("bad")
	}
	return true
}

func (c *Context) stmtSelect() bool {
	c.F("select {\n")
	for rand.Intn(5) != 0 {
		if rand.Intn(2) == 0 {
			t := c.aType(TraitSendable)
			c.F("case %v <- %v:\n", c.rvalue(t), c.rvalue(t.ktyp))
			c.block()
		} else {
			c.EnterScope()
			t := c.aType(TraitReceivable)
			ch := c.rvalue(t)
			switch c.choice("single", "double", "single_decl", "double_decl") {
			case "single":
				c.F("case %v = <-%v:\n", c.lvalue(t.ktyp), ch)
			case "double":
				c.F("case %v, %v = <-%v:\n", c.lvalue(t.ktyp), c.lvalue(c.boolType), ch)
			case "single_decl":
				vv := &Var{id: c.newId(), typ: t.ktyp}
				c.F("case %v := <-%v:\n", vv.id, ch)
				c.vars = append(c.vars, vv)
			case "double_decl":
				vv := &Var{id: c.newId(), typ: t.ktyp}
				ok := &Var{id: c.newId(), typ: c.boolType}
				c.F("case %v, %v := <-%v:\n", vv.id, ok.id, ch)
				c.vars = append(c.vars, vv)
				c.vars = append(c.vars, ok)
			default:
				panic("bad")
			}
			c.block()
			c.LeaveScope()
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
*/
