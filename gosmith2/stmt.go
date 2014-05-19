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
		//stmtSelect,
		stmtTypeDecl,
		stmtCall,
		stmtReturn,
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
	id := newId()
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
	enterBlock(true)
	// TODO: note that we are in for, to generate break/continue
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
				id := newId()
				line("for %v := range %v {", id, s)
				defineVar(id, intType)
			case "twoDecl":
				id := newId()
				id2 := newId()
				line("for %v, %v := range %v {", id, id2, s)
				defineVar(id, intType)
				defineVar(id2, t)
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
	genBlock()
	leaveBlock()
	line("}")
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
		vv := newId()
		ok := newId()
		line("%v, %v := <-%v", vv, ok, ch)
		defineVar(vv, t.ktyp)
		defineVar(ok, boolType)
	default:
		panic("bad")
	}
}

func stmtTypeDecl() {
	id := newId()
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
				vv := newId()
				line("case %v := <-%v:", vv, ch)
				defineVar(vv, elem)
			case "twoDecl":
				vv := newId()
				ok := newId()
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
}

func stmtCall() {
	if rndBool() {
		stmtCallBuiltin()
	}
	t := atype(ClassFunction)
	line("%v(%v)", rvalue(t), fmtRvalueList(t.styp))
}

func stmtCallBuiltin() {
	switch fn := choice("close", "copy", "delete", "panic", "print", "println", "recover"); fn {
	case "close":
	case "copy":
	case "delete":
	case "panic":
	case "print":
		fallthrough
	case "println":
		list := atypeList(TraitPrintable)
		line("%v(%v)", fn, fmtRvalueList(list))
	case "recover":
	default:
		panic("bad")
	}
}
