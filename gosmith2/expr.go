package main

import (
	"bytes"
	"fmt"
)

func initExpressions() {
	expressions = []func(res *Type) string{
		exprLiteral,
		exprVar,
		exprRecv,
		exprVar,
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
	return exprVar(t)
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
	// TODO: currently it triggers "internal compiler error: walkexpr ORECV" too frequently
	if true {
		return ""
	}
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
		return ""
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
		return ""
	case "imag":
		return ""
	case "make":
		return ""
	case "new":
		return ""
	case "real":
		return ""
	case "recover":
		return ""
	default:
		panic("bad")
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
