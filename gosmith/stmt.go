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
		stmtVarDecl,
		stmtCall,
		stmtReturn,
		stmtBreak,
		stmtContinue,
		stmtGoto,
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
	list := atypeList(TraitAny)
	str, vars := fmtOasVarList(list)
	line("%v := %v", str, fmtRvalueList(list))
	for _, v := range vars {
		defineVar(v.id, v.typ)
	}
}

func stmtReturn() {
	line("return %v", fmtRvalueList(curFunc.rets))
}

func stmtAs() {
	types := atypeList(TraitAny)
	line("%v = %v", fmtLvalueList(types), fmtRvalueList(types))
}

func stmtInc() {
	if rndBool() {
		m := exprIndexMap(atype(ClassNumeric))
		if m != "" {
			line("%v %v", m, choice("--", "++"))
			return
		}
	}
	line("%v %v", lvalue(atype(ClassNumeric)), choice("--", "++"))
}

func stmtIf() {
	enterBlock(true)
	enterBlock(true)
	if rndBool() {
		line("if %v {", rvalue(atype(ClassBoolean)))
	} else {
		line("if %v; %v {", stmtSimple(true, nil), rvalue(atype(ClassBoolean)))
	}
	genBlock()
	if rndBool() {
		line("} else {")
		genBlock()
	}
	leaveBlock()
	line("}")
	leaveBlock()
}

func stmtFor() {
	enterBlock(true)
	enterBlock(true)
	curBlock.isBreakable = true
	curBlock.isContinuable = true
	var vars []*Var
	switch choice("simple", "complex", "range") {
	case "simple":
		line("for %v {", rvalue(atype(ClassBoolean)))
	case "complex":
		line("for %v; %v; %v {", stmtSimple(true, nil), rvalue(atype(ClassBoolean)), stmtSimple(false, nil))
	case "range":
		switch choice("slice", "string", "channel", "map") {
		case "slice":
			t := atype(TraitAny)
			s := rvalue(sliceOf(t))
			switch choice("one", "two", "oneDecl", "twoDecl") {
			case "one":
				line("for %v = range %v {", lvalueOrBlank(intType), s)
			case "two":
				line("for %v, %v = range %v {", lvalueOrBlank(intType), lvalueOrBlank(t), s)
			case "oneDecl":
				id := newId("Var")
				line("for %v := range %v {", id, s)
				vars = append(vars, &Var{id: id, typ: intType})
			case "twoDecl":
				types := []*Type{intType, t}
				str := ""
				str, vars = fmtOasVarList(types)
				line("for %v := range %v {", str, s)
			default:
				panic("bad")
			}
		case "string":
			s := rvalue(stringType)
			switch choice("one", "two", "oneDecl", "twoDecl") {
			case "one":
				line("for %v = range %v {", lvalueOrBlank(intType), s)
			case "two":
				line("for %v, %v = range %v {", lvalueOrBlank(intType), lvalueOrBlank(runeType), s)
			case "oneDecl":
				id := newId("Var")
				line("for %v := range %v {", id, s)
				vars = append(vars, &Var{id: id, typ: intType})
			case "twoDecl":
				types := []*Type{intType, runeType}
				str := ""
				str, vars = fmtOasVarList(types)
				line("for %v := range %v {", str, s)
			default:
				panic("bad")
			}
		case "channel":
			cht := atype(ClassChan)
			ch := rvalue(cht)
			switch choice("one", "oneDecl") {
			case "one":
				line("for %v = range %v {", lvalue(cht.ktyp), ch)
			case "oneDecl":
				id := newId("Var")
				line("for %v := range %v {", id, ch)
				vars = append(vars, &Var{id: id, typ: cht.ktyp})
			default:
				panic("bad")
			}
		case "map":
			t := atype(ClassMap)
			m := rvalue(t)
			switch choice("one", "two", "oneDecl", "twoDecl") {
			case "one":
				line("for %v = range %v {", lvalue(t.ktyp), m)
			case "two":
				line("for %v, %v = range %v {", lvalue(t.ktyp), lvalue(t.vtyp), m)
			case "oneDecl":
				id := newId("Var")
				line("for %v := range %v {", id, m)
				vars = append(vars, &Var{id: id, typ: t.ktyp})
			case "twoDecl":
				types := []*Type{t.ktyp, t.vtyp}
				str := ""
				str, vars = fmtOasVarList(types)
				line("for %v := range %v {", str, m)
			default:
				panic("bad")
			}
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
	leaveBlock()
	line("}")
	leaveBlock()
}

func stmtSimple(oas bool, newVars *[]*Var) string {
	// We emit a fake statement in "oas", so make sure that nothing can be inserted in between.
	if curBlock.extendable {
		panic("bad")
	}
	// "send" crashes gccgo with random errors too frequently.
	// https://gcc.gnu.org/bugzilla/show_bug.cgi?id=61273
	switch choice("empty", "inc", "assign", "oas" /*"send",*/, "expr") {
	case "empty":
		return ""
	case "inc":
		return F("%v %v", lvalue(atype(ClassNumeric)), choice("--", "++"))
	case "assign":
		list := atypeList(TraitAny)
		return F("%v = %v", fmtLvalueList(list), fmtRvalueList(list))
	case "oas":
		if !oas {
			return ""
		}
		list := atypeList(TraitAny)
		str, vars := fmtOasVarList(list)
		if newVars != nil {
			*newVars = vars
		}
		res := F("%v := %v", str, fmtRvalueList(list))
		line("")
		for _, v := range vars {
			defineVar(v.id, v.typ)
		}
		return res
	case "send":
		t := atype(TraitSendable)
		return F("%v <- %v", rvalue(t), rvalue(t.ktyp))
	case "expr":
		return ""
	default:
		panic("bad")
	}
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
		vv := newId("Var")
		ok := newId("Var")
		line("%v, %v := <-%v", vv, ok, ch)
		defineVar(vv, t.ktyp)
		defineVar(ok, boolType)
	default:
		panic("bad")
	}
}

func stmtTypeDecl() {
	id := newId("Type")
	t := atype(TraitAny)
	line("type %v %v", id, t.id)

	newTyp := new(Type)
	*newTyp = *t
	newTyp.id = id
	newTyp.namedUserType = true
	if t.class == ClassStruct {
		newTyp.literal = func() string {
			// replace struct name with new type id
			l := t.literal()
			l = l[len(t.id)+1:]
			return "(" + id + l
		}
		newTyp.complexLiteral = func() string {
			// replace struct name with new type id
			l := t.complexLiteral()
			l = l[len(t.id)+1:]
			return "(" + id + l
		}
	} else {
		newTyp.literal = func() string {
			return F("%v(%v)", id, t.literal())
		}
		if t.complexLiteral != nil {
			newTyp.complexLiteral = func() string {
				return F("%v(%v)", id, t.complexLiteral())
			}
		}
	}
	defineType(newTyp)
}

func stmtVarDecl() {
	id := newId("Var")
	t := atype(TraitAny)
	line("var %v %v = %v", id, t.id, rvalue(t))
	defineVar(id, t)
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
				vv := newId("Var")
				line("case %v := <-%v:", vv, ch)
				defineVar(vv, elem)
			case "twoDecl":
				vv := newId("Var")
				ok := newId("Var")
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
	enterBlock(true)
	curBlock.isBreakable = true
	var vars []*Var
	if rndBool() {
		line("switch %v {", cond)
	} else {
		line("switch %v; %v {", stmtSimple(true, &vars), cond)
	}
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
	if fallth || len(vars) > 0 || rndBool() {
		enterBlock(true)
		line("default:")
		genBlock()
		for _, v := range vars {
			line("_ = %v", v.id)
			v.used = true
		}
		leaveBlock()
	}
	leaveBlock()
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

func stmtGoto() {
	// TODO: suppport goto down
	id := materializeGotoLabel()
	line("goto %v", id)
}

func stmtSink() {
	// Makes var escape.
	line("SINK = %v", exprVar(atype(TraitAny)))
}
