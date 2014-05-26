package main

import (
	"bytes"
	"fmt"
)

func initExpressions() {
	expressions = []func(res *Type) string{
		exprLiteral,
		exprVar,
		exprFunc,
		exprSelectorField,
		exprRecv,
		exprArith,
		exprEqual,
		exprOrder,
		exprCall,
		exprCallBuiltin,
		exprAddress,
		exprDeref,
		exprSlice,
		exprIndexSlice,
		exprIndexArray,
		exprIndexString,
		exprIndexMap,
		exprConversion,
	}
}

func expression(res *Type) string {
	exprCount++
	totalExprCount++
	if exprDepth >= NExprDepth || exprCount >= NExprCount || totalExprCount >= NTotalExprCount {
		return res.literal()
	}
	for {
		exprDepth++
		s := expressions[rnd(len(expressions))](res)
		exprDepth--
		if s != "" {
			return s
		}
	}
}

func rvalue(t *Type) string {
	return expression(t)
}

// rvalue, but not a const
// used to index arrays and strings
func nonconstRvalue(t *Type) string {
	if t.class != ClassNumeric {
		panic("bad")
	}
trying:
	for {
		res := ""
		switch choice("lvalue", "call", "len", "selector", "recv", "arith", "indexMap", "conv") {
		case "lvalue":
			res = lvalue(t)
		case "call":
			res = exprCall(t)
		case "len":
			tt := atype(TraitLenCapable)
			fn := choice("len", "cap")
			if (tt.class == ClassString || tt.class == ClassMap) && fn == "cap" {
				break
			}
			if tt.class == ClassArray {
				// len/cap are const
				break
			}
			res = F("(%v)((%v)(%v))", t.id, fn, lvalue(tt))
		case "selector":
			res = exprSelectorField(t)
		case "recv":
			res = exprRecv(t)
		case "arith":
			res = F("(%v) %v (%v)", lvalue(t), choice("+", "-"), rvalue(t))
		case "indexMap":
			res = exprIndexMap(t)
		case "conv":
			res = F("(%v)(%v %v)", t.id, lvalue(atype(ClassNumeric)), choice("", ","))
		default:
			panic("bad")
		}
		if res == "" {
			continue trying
		}
		return res
	}
}

func lvalue(t *Type) string {
	for {
		switch choice("var", "indexSlice", "indexArray", "selector", "deref") {
		case "var":
			return exprVar(t)
		case "indexSlice":
			return exprIndexSlice(t)
		case "indexArray":
			return F("(%v)[%v]", lvalue(arrayOf(t)), nonconstRvalue(intType))
		case "selector":
			for i := 0; i < 10; i++ {
				st := atype(ClassStruct)
				for _, e := range st.elems {
					if e.typ == t {
						return F("(%v).%v", lvalue(st), e.id)
					}
				}
			}
			continue
		case "deref":
			return exprDeref(t)
		default:
			panic("bad")
		}
	}
}

func lvalueOrBlank(t *Type) string {
	if rndBool() {
		return "_"
	}
	return lvalue(t)
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

func fmtOasVarList(list []*Type) (str string, newVars []*Var) {
	allVars := vars()
	var buf bytes.Buffer
	for i, t := range list {
		expr := "_"
		// First, try to find an existing var in the same scope.
		if rndBool() {
			for _, v := range allVars {
				if v.typ == t && v.block == curBlock {
					expr = v.id
					break
				}
			}
		}
		if rndBool() || (i == len(list)-1 && len(newVars) == 0) {
			expr = newId("Var")
			newVars = append(newVars, &Var{id: expr, typ: t})
		}

		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(expr)
	}
	return buf.String(), newVars
}

func exprLiteral(res *Type) string {
	if res.complexLiteral != nil {
		return res.complexLiteral()
	}
	return res.literal()
}

func exprVar(res *Type) string {
	for _, v := range vars() {
		if v.typ == res {
			return v.id
		}
	}
	return materializeVar(res)
}

func exprSelectorField(res *Type) string {
	for i := 0; i < 10; i++ {
		st := atype(ClassStruct)
		for _, e := range st.elems {
			if e.typ == res {
				return F("(%v).%v", rvalue(st), e.id)
			}
		}
	}
	return ""
}

func exprFunc(res *Type) string {
	if !satisfiesTrait(res, TraitGlobal) {
		return ""
	}
	var f *Func
	for _, f1 := range packages[curPackage].toplevFuncs {
		if len(f1.rets) == 1 && f1.rets[0] == res {
			f = f1
			break
		}
	}
	if f == nil {
		f = materializeFunc([]*Type{res})
	}
	if rndBool() {
		return F("%v(%v)", f.name, fmtRvalueList(f.args))
	} else {
		var f0 *Func
	loop:
		for _, f1 := range packages[curPackage].toplevFuncs {
			if len(f1.rets) == len(f.args) {
				for i := range f.args {
					// TODO: check assignability
					if f1.rets[i] != f.args[i] {
						continue loop
					}
				}
				f0 = f1
				break
			}
		}
		if f0 == nil {
			f0 = materializeFunc(f.args)
		}
		return F("%v(%v(%v))", f.name, f0.name, fmtRvalueList(f0.args))
	}
}

func exprAddress(res *Type) string {
	if res.class != ClassPointer {
		return ""
	}
	if res.ktyp.class == ClassStruct && rndBool() {
		return F("&%v", res.ktyp.complexLiteral())
	}
	return F("(%v)(&(%v))", res.id, lvalue(res.ktyp))
}

func exprDeref(res *Type) string {
	return F("(*(%v))", lvalue(pointerTo(res)))
}

func exprRecv(res *Type) string {
	t := chanOf(res)
	return F("(<- %v)", rvalue(t))
}

func exprArith(res *Type) string {
	if res.class != ClassNumeric && res.class != ClassComplex {
		return ""
	}
	// "/" causes division by zero
	// "*" causes generation of -1 index in int(real(1i * 1i))
	return F("(%v) + (%v)", rvalue(res), rvalue(res))
}

func exprEqual(res *Type) string {
	if res != boolType {
		return ""
	}
	t := atype(TraitComparable)
	return F("(%v) %v (%v)", rvalue(t), choice("==", "!="), rvalue(t))
}

func exprOrder(res *Type) string {
	if res != boolType {
		return ""
	}
	t := atype(TraitOrdered)
	return F("(%v) %v (%v)", rvalue(t), choice("<", "<=", ">", ">="), rvalue(t))

}

func exprCall(ret *Type) string {
	t := funcOf(atypeList(TraitAny), []*Type{ret})
	return F("%v(%v)", rvalue(t), fmtRvalueList(t.styp))
}

func exprCallBuiltin(ret *Type) string {
	switch fn := choice("append", "cap", "complex", "copy", "imag", "len", "make", "new", "real", "recover"); fn {
	case "append":
		if ret.class != ClassSlice {
			return ""
		}
		switch choice("one", "two", "slice") {
		case "one":
			return F("%v(%v, %v)", fn, rvalue(ret), rvalue(ret.ktyp))
		case "two":
			return F("%v(%v, %v, %v)", fn, rvalue(ret), rvalue(ret.ktyp), rvalue(ret.ktyp))
		case "slice":
			return F("%v(%v, %v...)", fn, rvalue(ret), rvalue(ret))
		default:
			panic("bad")
		}
	case "len", "cap":
		if ret != intType { // TODO: must be convertable
			return ""
		}
		t := atype(TraitLenCapable)
		if (t.class == ClassString || t.class == ClassMap) && fn == "cap" {
			return ""

		}
		return F("%v(%v)", fn, rvalue(t))
	case "copy":
		if ret != intType {
			return ""
		}
		return F("%v", exprCopySlice())
	case "make":
		if ret.class != ClassSlice && ret.class != ClassMap && ret.class != ClassChan {
			return ""
		}
		cap := ""
		if ret.class == ClassSlice {
			if rndBool() {
				cap = F(", %v", rvalue(intType))
			} else {
				// Careful to not generate "len larger than cap".
				cap = F(", 0, %v", rvalue(intType))
			}
		} else if rndBool() {
			cap = F(", %v", rvalue(intType))
		}
		return F("make(%v %v)", ret.id, cap)
	case "new":
		if ret.class != ClassPointer {
			return ""
		}
		return F("new(%v)", ret.ktyp.id)
	case "recover":
		if ret != efaceType {
			return ""
		}
		return "recover()"
	case "real", "imag":
		if ret == float32Type {
			return F("real(%v)", rvalue(complex64Type))
		}
		if ret == float64Type {
			return F("real(%v)", rvalue(complex128Type))
		}
		return ""
	case "complex":
		if ret == complex64Type {
			return F("complex(%v, %v)", rvalue(float32Type), rvalue(float32Type))
		}
		if ret == complex128Type {
			return F("complex(%v, %v)", rvalue(float64Type), rvalue(float64Type))
		}
		return ""
	default:
		panic("bad")
	}
}

func exprCopySlice() string {
	if rndBool() {
		t := atype(ClassSlice)
		return F("copy(%v, %v)", rvalue(t), rvalue(t))
	} else {
		return F("copy(%v, %v)", rvalue(sliceOf(byteType)), rvalue(stringType))
	}
}

func exprSlice(ret *Type) string {
	if ret.class != ClassSlice {
		return ""
	}
	i0 := ""
	if rndBool() {
		i0 = lvalue(intType)
	}
	i2 := ""
	if rndBool() {
		i2 = ":" + lvalue(intType)
	}
	i1 := ":"
	if rndBool() || i2 != "" {
		i1 = ":" + lvalue(intType)
	}
	return F("(%v)[%v%v%v]", rvalue(ret), i0, i1, i2)
}

func exprIndexSlice(ret *Type) string {
	return F("(%v)[%v]", rvalue(sliceOf(ret)), nonconstRvalue(intType))
}

func exprIndexString(ret *Type) string {
	if ret != byteType {
		return ""
	}
	return F("(%v)[%v]", rvalue(stringType), nonconstRvalue(intType))
}

func exprIndexArray(ret *Type) string {
	// TODO: also handle indexing of pointers to arrays
	return F("(%v)[%v]", rvalue(arrayOf(ret)), nonconstRvalue(intType))
}

func exprIndexMap(ret *Type) string {
	// TODO: figure out something better
	for i := 0; i < 10; i++ {
		t := atype(ClassMap)
		if t.vtyp == ret {
			return F("(%v)[%v]", rvalue(t), rvalue(t.ktyp))
		}
	}
	return ""
}

func exprConversion(ret *Type) string {
	if ret.class == ClassNumeric {
		return F("(%v)(%v %v)", ret.id, rvalue(atype(ClassNumeric)), choice("", ","))
	}
	if ret.class == ClassComplex {
		return F("(%v)(%v %v)", ret.id, rvalue(atype(ClassComplex)), choice("", ","))
	}
	if ret == stringType {
		switch choice("int", "byteSlice", "runeSlice") {
		case "int":
			// We produce a string of length at least 3, to not produce
			// "invalid string index 1 (out of bounds for 1-byte string)"
			return F("(%v)((%v) + (1<<24) %v)", ret.id, rvalue(intType), choice("", ","))
		case "byteSlice":
			return F("(%v)(%v %v)", ret.id, rvalue(sliceOf(byteType)), choice("", ","))
		case "runeSlice":
			return F("(%v)(%v %v)", ret.id, rvalue(sliceOf(runeType)), choice("", ","))
		default:
			panic("bad")
		}
	}
	if ret.class == ClassSlice && (ret.ktyp == byteType || ret.ktyp == runeType) {
		return F("(%v)(%v %v)", ret.id, rvalue(stringType), choice("", ","))
	}
	// TODO: handle "x is assignable to T"
	// TODO: handle "x's type and T have identical underlying types"
	// TODO: handle "x's type and T are unnamed pointer types and their pointer base types have identical underlying types"
	return ""
}
