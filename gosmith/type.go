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
)

type TypeTrait int

const (
	TraitAny TypeTrait = iota
	TraitOrdered
	TraitComparable
	TraitIndexable
	TraitReceivable
)

type Type struct {
	id      Id
	class   TypeClass
	ktyp    *Type   // map key, chan elem, array elem, slice elem, pointee type
	vtyp    *Type   // map val
	utyp    *Type   // underlying type
	rtyp    []*Type // function return types, struct elements
	atyp    []*Type // function argument types
	literal func() string
}

func (c *Context) initTypes() {
	c.types = []*Type{
		&Type{id: "bool", class: ClassBoolean, literal: func() string { return "false" }},
		&Type{id: "int", class: ClassNumeric, literal: func() string { return "1" }},
		&Type{id: "uint", class: ClassNumeric, literal: func() string { return "uint(1)" }},
		&Type{id: "uintptr", class: ClassNumeric, literal: func() string { return "uintptr(0)" }},
		&Type{id: "int16", class: ClassNumeric, literal: func() string { return "int16(1)" }},
		&Type{id: "float64", class: ClassNumeric, literal: func() string { return "1.1" }},
		&Type{id: "string", class: ClassString, literal: func() string { return "\"foo\"" }},
	}
	for _, t := range c.types {
		t.utyp = t
	}
	c.boolType = c.types[0]
}

func (c *Context) aType(trait TypeTrait) *Type {
	for {
		if c.rand(3) == 0 {
			var cand []*Type
			for _, t := range c.types {
				if satisfiesTrait(t, trait) {
					cand = append(cand, t)
				}
			}
			if len(cand) > 0 {
				return cand[c.rand(len(cand))]
			}
		}
		t := c.typeLit(trait)
		if t != nil {
			return t
		}
	}
}

func (c *Context) aTypeList() []*Type {
	n := c.rand(4) + 1
	list := make([]*Type, n)
	for i := 0; i < n; i++ {
		list[i] = c.aType(TraitAny)
	}
	return list
}

func (c *Context) typeLit(trait TypeTrait) *Type {
	switch c.rand(8) {
	case 0: // ArrayType
		if trait != TraitAny && trait != TraitIndexable && trait != TraitComparable {
			return nil
		}
		elemTrait := TraitAny
		if trait == TraitComparable {
			elemTrait = TraitComparable
		}
		elem := c.aType(elemTrait)
		size := c.rand(10)
		return &Type{id: Id(fmt.Sprintf("[%v]%v", size, elem.id)), class: ClassArray, ktyp: elem, literal: func() string {
			var buf bytes.Buffer
			fmt.Fprintf(&buf, "[%v]%v{", size, elem.id)
			for i := 0; i < size; i++ {
				if i != 0 {
					fmt.Fprintf(&buf, ",")
				}
				fmt.Fprintf(&buf, "%v", c.lvalue(elem))
			}
			fmt.Fprintf(&buf, "}")
			return buf.String()
		}}
	case 1: // StructType
		return nil
	case 2: // PointerType
		return nil
	case 3: // FunctionType
		return nil
	case 4: // InterfaceType
		return nil
	case 5: // SliceType
		return nil
	case 6: // MapType
		return nil
	case 7: // ChannelType
		return nil
	default:
		panic("bad")
	}
}

func satisfiesTrait(t *Type, trait TypeTrait) bool {
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
	default:
		panic("bad")
	}
}
