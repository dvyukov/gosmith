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
		exprAddress,
		exprDeref,
		exprSlice,
		exprIndexSlice,
		exprIndexArray,
		exprIndexString,
		exprIndexMap,
	}
}

func expression(res *Type) string {
	exprCount++
	totalExprCount++
	if exprDepth >= NExprDepth || exprCount >= NExprCount || totalExprCount >= NTotalExprCount {
		return exprLiteral(res)
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

func lvalue(t *Type) string {
	switch choice("var", "index", "deref") {
	case "var":
		return exprVar(t)
	case "index":
		return exprIndexSlice(t)
	case "deref":
		return exprDeref(t)
	default:
		panic("bad")
	}
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
	for _, v := range vars() {
		if v.typ == res {
			return v.id
		}
	}
	return materializeVar(res)
}

func exprSelectorField(res *Type) string {
	return ""
}

func exprFunc(res *Type) string {
	if !satisfiesTrait(res, TraitGlobal) {
		return ""
	}
	var f *Func
	for _, f = range toplevFuncs {
		if len(f.rets) == 1 && f.rets[0] == res {
			break
		}
	}
	if f == nil {
		f = materializeFunc(res)
	}
	return F("%v(%v)", f.name, fmtRvalueList(f.args))
}

func exprAddress(res *Type) string {
	if res.class != ClassPointer {
		return ""
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
	if res.class != ClassNumeric {
		return ""
	}
	return F("(%v) %v (%v)", rvalue(res), choice("+", "*"), rvalue(res))
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
	if rndBool() {
		return exprCallBuiltin(ret)
	}
	return ""
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
	case "cap":
		fallthrough
	case "len":
		if ret != intType { // TODO: must be convertable
			return ""
		}
		t := atype(TraitLenCapable)
		if (t.class == ClassString || t.class == ClassMap) && fn == "cap" {
			return ""

		}
		return F("%v(%v)", fn, rvalue(t))
	case "complex":
		return ""
	case "copy":
		if ret != intType {
			return ""
		}
		return F("%v", exprCopySlice())
	case "imag":
		return ""
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
	case "real":
		return ""
	case "recover":
		if ret != efaceType {
			return ""
		}
		return "recover()"
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
	return F("(%v)[%v]", rvalue(sliceOf(ret)), rvalue(intType))
}

func exprIndexString(ret *Type) string {
	// TODO: currently generates too many out-of-bounds errors for string literals
	if true {
		return ""
	}
	if ret != byteType {
		return ""
	}
	return F("(%v)[%v]", rvalue(stringType), rvalue(intType))
}

func exprIndexArray(ret *Type) string {
	// TODO: also handle indexing of pointers to arrays
	return ""
}

func exprIndexMap(ret *Type) string {
	return ""
}
