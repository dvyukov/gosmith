package main

import (
	"bytes"
)

type Type struct {
	id    string
	ktyp  *Type  // map key, array elem, slice elem, pointee type
	vtyp  *Type  // map val
	elems []*Var // struct fileds
	vars  []*Var // all existing vars of this type
	lit   func(t *Type) (string, int)
}

type Var struct {
	id  string
	typ *Type
	len int // string/slice/array len
	val string
}

type Trait int

const (
	Any Trait = iota
	Hashable
)

var (
	vars  []*Var
	types []*Type
	idSeq int
)

func atype(tr Trait) *Type {
	if rndBool() {
		return types[rnd(len(types))]
	} else {
		t := newType(tr)
		types = append(types, t)
		return t
	}
}

func newType(tr Trait) *Type {
retry:
	switch choice("slice" /*, "array"*/) {
	case "slice":
		if tr == Hashable {
			goto retry
		}
		elem := atype(Any)
		return &Type{id: f("[]%v", elem.id), lit: func(t *Type) (string, int) {
			n := rnd(5)
      if n == 0 {
        return f("([]%v)(nil)", elem.id), n
      } else {
        return f("[]%v{%v}", elem.id, varList(repeatType(elem, n))), n
      }
		}}

	case "array":
		panic("bad")
	default:
		panic("bad")
	}
	/*
		return &Type{id: "interface{}", lit: func(t *Type) (string, int) {
			if rndBool() && len(vars) > 0 {
				v := vars[rnd(len(vars))]
				return v.id, 0
			} else {
				t1 := atype(tr)
				return t1.lit(t1)
			}
		}}
	*/
}

func repeatType(t *Type, n int) []*Type {
	types := make([]*Type, n)
	for i := 0; i < n; i++ {
		types[i] = t
	}
	return types
}

func varList(types []*Type) string {
	var buf bytes.Buffer
	for i, t := range types {
		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(newVar(t))
	}
	return buf.String()
}

func newVar(t *Type) string {
	idSeq++
	id := f("v%v", idSeq)
	val, ln := "", 0
	if rndBool() && len(t.vars) > 0 {
		v := t.vars[rnd(len(t.vars))]
		val, ln = v.id, v.len
	} else {
		val, ln = t.lit(t)
	}
	v := &Var{id: id, typ: t, val: val, len: ln}
	t.vars = append(t.vars, v)
	vars = append(vars, v)
	return id
}

func sliceExistingVar(t *Type, full bool) (string, int) {
	v := t.vars[rnd(len(t.vars))]
	i0 := rnd(v.len + 1)
	i1 := i0 + rnd(rnd(v.len+1-i0))
	i2 := i1 + rnd(rnd(v.len+1-i1))
	if full {
		return f("%v[%v:%v:%v]", v.id, i0, i1, i2), i1 - i0
	} else {
		return f("%v[%v:%v]", v.id, i0, i1), i1 - i0
	}
}

func unleashMonkeys() {
	types = []*Type{
		&Type{id: "int", lit: func(t *Type) (string, int) {
			return "0", 0
		}},
		&Type{id: "string", lit: func(t *Type) (string, int) {
			if rndBool() && len(t.vars) > 0 {
				return sliceExistingVar(t, false)
			} else {
				ln := rnd(32)
				return f("string(([]byte(`0123456789012345678901`))[:%v])", ln), ln
			}
		}},
	}
	newVar(atype(Any))
}

/*
	switch choice("array", "struct", "pointer", "interface", "slice", "map") {

	case TraitHashable:
		if t.class == ClassFunction || t.class == ClassMap || t.class == ClassSlice {
		if t.class == ClassArray && !satisfiesTrait(t.ktyp, TraitHashable) {
		if t.class == ClassStruct {
			for _, e := range t.elems {
				if !satisfiesTrait(e.typ, TraitHashable) {
					return false

		exprLiteral,
		exprSelectorField,
		exprAddress,
		exprSlice,
		exprIndexSlice,
		exprIndexArray,
		exprIndexString,
*/
