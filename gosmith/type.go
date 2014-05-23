package main

import (
	"bytes"
	"fmt"
)

type TypeClass int

const (
	ClassBoolean TypeClass = iota
	ClassNumeric
	ClassString
	ClassArray
	ClassSlice
	ClassStruct
	ClassPointer
	ClassFunction
	ClassInterface
	ClassMap
	ClassChan

	TraitAny
	TraitOrdered
	TraitComparable
	TraitIndexable
	TraitReceivable
	TraitSendable
	TraitHashable
	TraitPrintable
	TraitLenCapable
	TraitGlobal
)

type Type struct {
	id             string
	class          TypeClass
	ktyp           *Type   // map key, chan elem, array elem, slice elem, pointee type
	vtyp           *Type   // map val
	utyp           *Type   // underlying type
	styp           []*Type // function arguments
	rtyp           []*Type // function return values
	elems          []*Var  // struct fileds and interface methods
	literal        func() string
	complexLiteral func() string

	// TODO: cache types
	// pointerTo *Type
}

func initTypes() {
	predefinedTypes = []*Type{
		&Type{id: "string", class: ClassString, literal: func() string { return "\"foo\"" }},
		&Type{id: "bool", class: ClassBoolean, literal: func() string { return "false" }},
		&Type{id: "int", class: ClassNumeric, literal: func() string { return "1" }},
		&Type{id: "byte", class: ClassNumeric, literal: func() string { return "byte(0)" }},
		&Type{id: "interface{}", class: ClassInterface, literal: func() string { return "interface{}(nil)" }},
		&Type{id: "rune", class: ClassNumeric, literal: func() string { return "rune(0)" }},
		&Type{id: "uint", class: ClassNumeric, literal: func() string { return "uint(1)" }},
		&Type{id: "uintptr", class: ClassNumeric, literal: func() string { return "uintptr(0)" }},
		&Type{id: "int16", class: ClassNumeric, literal: func() string { return "int16(1)" }},
		&Type{id: "float64", class: ClassNumeric, literal: func() string { return "1.0" }},
		&Type{id: "float32", class: ClassNumeric, literal: func() string { return "float32(1.0)" }},
		&Type{id: "error", class: ClassInterface, literal: func() string { return "error(nil)" }},
	}
	for _, t := range predefinedTypes {
		t.utyp = t
	}

	stringType = predefinedTypes[0]
	boolType = predefinedTypes[1]
	intType = predefinedTypes[2]
	byteType = predefinedTypes[3]
	efaceType = predefinedTypes[4]
	runeType = predefinedTypes[5]

	stringType.complexLiteral = func() string {
		if rndBool() {
			return `"ab\x0acd"`
		}
		return "`abc\\x0acd`"
	}
}

func fmtTypeList(list []*Type, parens bool) string {
	var buf bytes.Buffer
	if parens || len(list) > 1 {
		buf.Write([]byte{'('})
	}
	for i, t := range list {
		if i != 0 {
			buf.Write([]byte{','})
		}
		fmt.Fprintf(&buf, "%v", t.id)
	}
	if parens || len(list) > 1 {
		buf.Write([]byte{')'})
	}
	return buf.String()
}

func atype(trait TypeClass) *Type {
	typeDepth++
	defer func() {
		typeDepth--
	}()
	for {
		if typeDepth >= 3 || rndBool() {
			var cand []*Type
			for _, t := range types() {
				if satisfiesTrait(t, trait) {
					cand = append(cand, t)
				}
			}
			if len(cand) > 0 {
				return cand[rnd(len(cand))]
			}
		}
		t := typeLit()
		if t != nil && satisfiesTrait(t, trait) {
			return t
		}
	}
}

func typeLit() *Type {
	switch choice("array", "chan", "struct", "pointer", "interface", "slice", "function", "map") {
	case "array":
		return arrayOf(atype(TraitAny))
	case "chan":
		return chanOf(atype(TraitAny))
	case "struct":
		var elems []*Var
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "struct { ")
		for rndBool() {
			e := &Var{id: newId("Field"), typ: atype(TraitAny)}
			elems = append(elems, e)
			fmt.Fprintf(&buf, "%v %v\n", e.id, e.typ.id)
		}
		fmt.Fprintf(&buf, "}")
		id := buf.String()
		return &Type{
			id:    id,
			class: ClassStruct,
			elems: elems,
			literal: func() string {
				return F("%v{}", id)
			},
			complexLiteral: func() string {
				if rndBool() {
					// unnamed
					var buf bytes.Buffer
					fmt.Fprintf(&buf, "%v{", id)
					for i := 0; i < len(elems); i++ {
						fmt.Fprintf(&buf, "%v, ", rvalue(elems[i].typ))
					}
					fmt.Fprintf(&buf, "}")
					return buf.String()
				} else {
					// named
					var buf bytes.Buffer
					fmt.Fprintf(&buf, "%v{", id)
					for i := 0; i < len(elems); i++ {
						if rndBool() {
							fmt.Fprintf(&buf, "%v: %v, ", elems[i].id, rvalue(elems[i].typ))
						}
					}
					fmt.Fprintf(&buf, "}")
					return buf.String()
				}
			},
		}
	case "pointer":
		return pointerTo(atype(TraitAny))
	case "interface":
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "interface { ")
		for rndBool() {
			fmt.Fprintf(&buf, " %v %v %v\n", newId("Method"), fmtTypeList(atypeList(TraitAny), true), fmtTypeList(atypeList(TraitAny), false))
		}
		fmt.Fprintf(&buf, "}")
		return &Type{
			id:    buf.String(),
			class: ClassInterface,
			literal: func() string {
				return F("%v(nil)", buf.String())
			},
		}
	case "slice":
		return sliceOf(atype(TraitAny))
	case "function":
		return funcOf(atypeList(TraitAny), atypeList(TraitAny))
	case "map":
		ktyp := atype(TraitHashable)
		vtyp := atype(TraitAny)
		return &Type{
			id:    F("map[%v]%v", ktyp.id, vtyp.id),
			class: ClassMap,
			ktyp:  ktyp,
			vtyp:  vtyp,
			literal: func() string {
				if rndBool() {
					cap := ""
					if rndBool() {
						cap = "," + rvalue(intType)
					}
					return F("make(map[%v]%v %v)", ktyp.id, vtyp.id, cap)
				} else {
					return F("map[%v]%v{}", ktyp.id, vtyp.id)
				}
			},
		}
	default:
		panic("bad")
	}
}

func satisfiesTrait(t *Type, trait TypeClass) bool {
	if trait < TraitAny {
		return t.class == trait
	}

	switch trait {
	case TraitAny:
		return true
	case TraitOrdered:
		return t.class == ClassNumeric || t.class == ClassString
	case TraitComparable:
		return t.class == ClassBoolean || t.class == ClassNumeric || t.class == ClassString ||
			t.class == ClassPointer || t.class == ClassChan || t.class == ClassInterface
	case TraitIndexable:
		return t.class == ClassArray || t.class == ClassSlice || t.class == ClassString ||
			t.class == ClassMap
	case TraitReceivable:
		return t.class == ClassChan
	case TraitSendable:
		return t.class == ClassChan
	case TraitHashable:
		if t.class == ClassFunction || t.class == ClassMap || t.class == ClassSlice {
			return false
		}
		if t.class == ClassArray && !satisfiesTrait(t.ktyp, TraitHashable) {
			return false
		}
		if t.class == ClassStruct {
			for _, e := range t.elems {
				if !satisfiesTrait(e.typ, TraitHashable) {
					return false
				}
			}
		}
		return true
	case TraitPrintable:
		return t.class == ClassBoolean || t.class == ClassNumeric || t.class == ClassString ||
			t.class == ClassPointer || t.class == ClassInterface
	case TraitLenCapable:
		return t.class == ClassString || t.class == ClassSlice || t.class == ClassArray ||
			t.class == ClassMap || t.class == ClassChan
	case TraitGlobal:
		for _, t1 := range predefinedTypes {
			if t == t1 {
				return true
			}
		}
		return false
	default:
		panic("bad")
	}
}

func atypeList(trait TypeClass) []*Type {
	n := rnd(4) + 1
	list := make([]*Type, n)
	for i := 0; i < n; i++ {
		list[i] = atype(trait)
	}
	return list
}

func typeList(t *Type, n int) []*Type {
	list := make([]*Type, n)
	for i := 0; i < n; i++ {
		list[i] = t
	}
	return list
}

func pointerTo(elem *Type) *Type {
	return &Type{
		id:    F("*%v", elem.id),
		class: ClassPointer,
		ktyp:  elem,
		literal: func() string {
			return F("(*%v)(nil)", elem.id)
		}}
}

func chanOf(elem *Type) *Type {
	return &Type{
		id:    F("chan %v", elem.id),
		class: ClassChan,
		ktyp:  elem,
		literal: func() string {
			cap := ""
			if rndBool() {
				cap = "," + rvalue(intType)
			}
			return F("make(chan %v %v)", elem.id, cap)
		},
	}
}

func sliceOf(elem *Type) *Type {
	return &Type{
		id:    F("[]%v", elem.id),
		class: ClassSlice,
		ktyp:  elem,
		literal: func() string {
			return F("[]%v{}", elem.id)
		},
		complexLiteral: func() string {
			switch choice("normal", "keyed") {
			case "normal":
				return F("[]%v{%v}", elem.id, fmtRvalueList(typeList(elem, rnd(3))))
			case "keyed":
				n := rnd(3)
				var indexes []int
			loop:
				for len(indexes) < n {
					i := rnd(10)
					for _, i1 := range indexes {
						if i1 == i {
							continue loop
						}
					}
					indexes = append(indexes, i)
				}
				var buf bytes.Buffer
				fmt.Fprintf(&buf, "[]%v{", elem.id)
				for i, idx := range indexes {
					if i != 0 {
						fmt.Fprintf(&buf, ",")
					}
					fmt.Fprintf(&buf, "%v: %v", idx, rvalue(elem))
				}
				fmt.Fprintf(&buf, "}")
				return buf.String()
			default:
				panic("bad")
			}
		},
	}
}

func arrayOf(elem *Type) *Type {
	size := rnd(3)
	return &Type{
		id:    F("[%v]%v", size, elem.id),
		class: ClassArray,
		ktyp:  elem,
		literal: func() string {
			return F("[%v]%v{}", size, elem.id)
		},
		complexLiteral: func() string {
			switch choice("normal", "keyed") {
			case "normal":
				return F("[%v]%v{%v}", choice(F("%v", size), "..."), elem.id, fmtRvalueList(typeList(elem, size)))
			case "keyed":
				var buf bytes.Buffer
				fmt.Fprintf(&buf, "[%v]%v{", size, elem.id)
				for i := 0; i < size; i++ {
					if i != 0 {
						fmt.Fprintf(&buf, ",")
					}
					fmt.Fprintf(&buf, "%v: %v", i, rvalue(elem))
				}
				fmt.Fprintf(&buf, "}")
				return buf.String()
			default:
				panic("bad")
			}
		},
	}
}

func funcOf(alist, rlist []*Type) *Type {
	return &Type{
		id:    F("func%v %v", fmtTypeList(alist, true), fmtTypeList(rlist, false)),
		class: ClassFunction,
		styp:  alist,
		rtyp:  rlist,
		literal: func() string {
			return F("((func%v %v)(nil))", fmtTypeList(alist, true), fmtTypeList(rlist, false))
		},
	}
}

func dependsOn(t, t0 *Type) bool {
	if t == nil {
		return false
	}
	if t.class == ClassInterface {
		// We don't know how to walk all types referenced by an interface yet.
		return true
	}
	if t == t0 {
		return true
	}
	if dependsOn(t.ktyp, t0) {
		return true
	}
	if dependsOn(t.vtyp, t0) {
		return true
	}
	if dependsOn(t.ktyp, t0) {
		return true
	}
	for _, t1 := range t.styp {
		if dependsOn(t1, t0) {
			return true
		}
	}
	for _, t1 := range t.rtyp {
		if dependsOn(t1, t0) {
			return true
		}
	}
	for _, e := range t.elems {
		if dependsOn(e.typ, t0) {
			return true
		}
	}
	return false
}
