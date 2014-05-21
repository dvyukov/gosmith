package main

import (
	_ "fmt"
)

func initStatements() {
	statements = []func(){
		stmtOas,
		stmtAs,
		stmtInc,
		stmtIf,
		stmtFor,
		stmtSend,
		stmtRecv,
		stmtSelect,
		stmtSwitchExpr,
		stmtSwitchType,
		stmtTypeDecl,
		stmtCall,
		stmtReturn,
		stmtBreak,
		stmtContinue,
		stmtSink,
	}
}

func genStatement() {
	if stmtCount >= NStatements {
		return
	}
	exprCount = 0
	stmtCount++
	statements[rnd(len(statements))]()
}

func stmtOas() {
	id := newId("var")
	t := atype(TraitAny)
	line("%v := %v", id, rvalue(t))
	defineVar(id, t)
}

func stmtReturn() {
	line("return %v", fmtRvalueList(curFunc.rets))
}

func stmtAs() {
	types := atypeList(TraitAny)
	line("%v = %v", fmtLvalueList(types), fmtRvalueList(types))
}

func stmtInc() {
	line("%v %v", lvalue(atype(ClassNumeric)), choice("--", "++"))
}

func stmtIf() {
	enterBlock(true)
	line("if %v {", rvalue(atype(ClassBoolean)))
	genBlock()
	if rndBool() {
		line("} else {")
		genBlock()
	}
	line("}")
	leaveBlock()
}

func stmtFor() {
	// TODO: note that we are in for, to generate break/continue
	enterBlock(true)
	curBlock.isBreakable = true
	curBlock.isContinuable = true
	var vars []*Var
	switch choice("simple", "complex", "range") {
	case "simple":
		line("for %v {", rvalue(atype(ClassBoolean)))
	case "complex":
		line("for %v; %v; %v {", stmtSimple(), rvalue(atype(ClassBoolean)), stmtSimple())
	case "range":
		switch choice("slice" /*, "string", "channel", "map"*/) {
		case "slice":
			t := atype(TraitAny)
			s := rvalue(sliceOf(t))
			// TODO: handle _
			switch choice("one", "two", "oneDecl", "twoDecl") {
			case "one":
				line("for %v = range %v {", lvalue(intType), s)
			case "two":
				line("for %v, %v = range %v {", lvalue(intType), lvalue(t), s)
			case "oneDecl":
				id := newId("var")
				line("for %v := range %v {", id, s)
				vars = append(vars, &Var{id: id, typ: intType})
			case "twoDecl":
				id := newId("var")
				id2 := newId("var")
				line("for %v, %v := range %v {", id, id2, s)
				vars = append(vars, &Var{id: id, typ: intType})
				vars = append(vars, &Var{id: id2, typ: t})
			default:
				panic("bad")
			}
		case "string":
		case "channel":
		case "map":
		default:
			panic("bad")
		}
	default:
		panic("bad")
	}
	enterBlock(true)
	if len(vars) > 0 {
		line("")
		for _, v := range vars {
			defineVar(v.id, v.typ)
		}
	}
	genBlock()
	leaveBlock()
	line("}")
	leaveBlock()
}

func stmtSimple() string {
	// TODO: unimplemented
	return F("%v %v", lvalue(atype(ClassNumeric)), choice("--", "++"))
}

func stmtSend() {
	t := atype(TraitSendable)
	line("%v <- %v", rvalue(t), rvalue(t.ktyp))
}

func stmtRecv() {
	t := atype(TraitReceivable)
	ch := rvalue(t)
	switch choice("normal", "decl") {
	case "normal":
		line("%v, %v = <-%v", lvalue(t.ktyp), lvalue(boolType), ch)
	case "decl":
		vv := newId("var")
		ok := newId("var")
		line("%v, %v := <-%v", vv, ok, ch)
		defineVar(vv, t.ktyp)
		defineVar(ok, boolType)
	default:
		panic("bad")
	}
}

func stmtTypeDecl() {
	id := newId("type")
	t := atype(TraitAny)
	line("type %v %v", id, t.id)

	newTyp := new(Type)
	*newTyp = *t
	newTyp.id = id
	newTyp.literal = func() string {
		return F("%v(%v)", id, t.literal())
	}
	defineType(newTyp)
}

func stmtSelect() {
	enterBlock(true)
	line("select {")
	for rnd(5) != 0 {
		enterBlock(true)
		elem := atype(TraitAny)
		cht := chanOf(elem)
		ch := rvalue(cht)
		if rndBool() {
			line("case %v <- %v:", ch, rvalue(elem))
		} else {
			switch choice("one", "two", "oneDecl", "twoDecl") {
			case "one":
				line("case %v = <-%v:", lvalue(elem), ch)
			case "two":
				line("case %v, %v = <-%v:", lvalue(elem), lvalue(boolType), ch)
			case "oneDecl":
				vv := newId("var")
				line("case %v := <-%v:", vv, ch)
				defineVar(vv, elem)
			case "twoDecl":
				vv := newId("var")
				ok := newId("var")
				line("case %v, %v := <-%v:", vv, ok, ch)
				defineVar(vv, elem)
				defineVar(ok, boolType)
			default:
				panic("bad")
			}
		}
		genBlock()
		leaveBlock()
	}
	if rndBool() {
		enterBlock(true)
		line("default:")
		genBlock()
		leaveBlock()
	}
	line("}")
	leaveBlock()
}

func stmtSwitchExpr() {
	var t *Type
	cond := ""
	if rndBool() {
		t = atype(TraitComparable)
		cond = rvalue(t)
	} else {
		t = boolType
	}
	enterBlock(true)
	curBlock.isBreakable = true
	line("switch %v {", cond)
	// TODO: we generate at most one case, because if we generate more,
	// we can generate two cases with equal constants.
	fallth := false
	if rndBool() {
		enterBlock(true)
		line("case %v:", rvalue(t))
		genBlock()
		leaveBlock()
		if rndBool() {
			fallth = true
			line("fallthrough")
		}
	}
	if fallth || rndBool() {
		enterBlock(true)
		line("default:")
		genBlock()
		leaveBlock()
	}
	line("}")
	leaveBlock()
}

func stmtSwitchType() {
	cond := lvalue(atype(TraitAny))
	enterBlock(true)
	curBlock.isBreakable = true
	line("switch COND := (interface{})(%v); COND.(type) {", cond)
	if rndBool() {
		enterBlock(true)
		line("case %v:", atype(TraitAny).id)
		genBlock()
		leaveBlock()
	}
	if rndBool() {
		enterBlock(true)
		line("default:")
		genBlock()
		leaveBlock()
	}
	line("}")
	leaveBlock()
}

func stmtCall() {
	if rndBool() {
		stmtCallBuiltin()
	}
	t := atype(ClassFunction)
	prefix := choice("", "go", "defer")
	line("%v %v(%v)", prefix, rvalue(t), fmtRvalueList(t.styp))
}

func stmtCallBuiltin() {
	prefix := choice("", "go", "defer")
	switch fn := choice("close", "copy", "delete", "panic", "print", "println", "recover"); fn {
	case "close":
		line("%v %v(%v)", prefix, fn, rvalue(atype(ClassChan)))
	case "copy":
		line("%v %v", prefix, exprCopySlice())
	case "delete":
		t := atype(ClassMap)
		line("%v %v(%v, %v)", prefix, fn, rvalue(t), rvalue(t.ktyp))
	case "panic":
		line("%v %v(%v)", prefix, fn, rvalue(atype(TraitAny)))
	case "print":
		fallthrough
	case "println":
		list := atypeList(TraitPrintable)
		line("%v %v(%v)", prefix, fn, fmtRvalueList(list))
	case "recover":
		line("%v %v()", prefix, fn)
	default:
		panic("bad")
	}
}

func stmtBreak() {
	if !curBlock.isBreakable {
		return
	}
	line("break")
}

func stmtContinue() {
	if !curBlock.isContinuable {
		return
	}
	line("continue")
}

func stmtSink() {
	line("SINK = %v", exprVar(atype(TraitAny)))
}
