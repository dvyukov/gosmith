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
	TraitSendable
	TraitHashable
	TraitPrintable
	TraitLenCapable
	TraitFunction
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
	c.intType = c.types[1]
}

func (c *Context) aType(trait TypeTrait) *Type {
	c.typeDepth++
	defer func() {
		c.typeDepth--
	}()
	for {
		if c.typeDepth >= 3 || c.rand(2) == 0 {
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
		t := c.typeLit()
		if t != nil && satisfiesTrait(t, trait) {
			return t
		}
	}
}

func (c *Context) aTypeList(trait TypeTrait) []*Type {
	n := c.rand(4) + 1
	list := make([]*Type, n)
	for i := 0; i < n; i++ {
		list[i] = c.aType(trait)
	}
	return list
}

func (c *Context) typeLit() *Type {
	switch c.rand(8) {
	case 0: // ArrayType
		elem := c.aType(TraitAny)
		size := c.rand(3)
		return &Type{
			id:    Id(fmt.Sprintf("[%v]%v", size, elem.id)),
			class: ClassArray,
			ktyp:  elem,
			literal: func() string {
				var buf bytes.Buffer
				fmt.Fprintf(&buf, "[%v]%v{", size, elem.id)
				for i := 0; i < size; i++ {
					if i != 0 {
						fmt.Fprintf(&buf, ",")
					}
					fmt.Fprintf(&buf, "%v", c.rvalue(elem))
				}
				fmt.Fprintf(&buf, "}")
				return buf.String()
			}}
	case 1: // StructType
		return nil
	case 2: // PointerType
		return nil
	case 3: // FunctionType
		rlist := c.aTypeList(TraitAny)
		alist := c.aTypeList(TraitAny)
		return &Type{
			id:    Id(fmt.Sprintf("func%v %v", c.formatTypeList(alist, true), c.formatTypeList(rlist, false))),
			class: ClassFunction,
			rtyp:  rlist,
			atyp:  alist,
			literal: func() string {
				return fmt.Sprintf("((func%v %v)(nil))", c.formatTypeList(alist, true), c.formatTypeList(rlist, false))
			}}
		return nil
	case 4: // InterfaceType
		return nil
	case 5: // SliceType
		elem := c.aType(TraitAny)
		return c.sliceOf(elem)
	case 6: // MapType
		ktyp := c.aType(TraitHashable)
		vtyp := c.aType(TraitAny)
		return &Type{
			id:    Id(fmt.Sprintf("map[%v]%v", ktyp.id, vtyp.id)),
			class: ClassMap,
			ktyp:  ktyp,
			vtyp:  vtyp,
			literal: func() string {
				if c.rand(2) == 0 {
					cap := ""
					if c.rand(2) == 0 {
						cap = "," + c.rvalue(c.intType)
					}
					return fmt.Sprintf("make(map[%v]%v %v)", ktyp.id, vtyp.id, cap)
				} else {
					return fmt.Sprintf("map[%v]%v{}", ktyp.id, vtyp.id)
				}
			},
		}
	case 7: // ChannelType
		elem := c.aType(TraitAny)
		return c.chanOf(elem)
	default:
		panic("bad")
	}
}

func (c *Context) formatTypeList(list []*Type, parens bool) string {
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

func (c *Context) formatRvalueList(list []*Type) string {
	var buf bytes.Buffer
	for i, t := range list {
		if i != 0 {
			buf.Write([]byte{','})
		}
		buf.WriteString(c.rvalue(t))
	}
	return buf.String()
}

func (c *Context) formatLvalueList(list []*Type) string {
	var buf bytes.Buffer
	for i, t := range list {
		if i != 0 {
			buf.Write([]byte{','})
		}
		buf.WriteString(c.lvalue(t))
	}
	return buf.String()
}

func (c *Context) chanOf(elem *Type) *Type {
	return &Type{
		id:    Id(fmt.Sprintf("chan %v", elem.id)),
		class: ClassChan,
		ktyp:  elem,
		literal: func() string {
			cap := ""
			if c.rand(2) == 0 {
				cap = "," + c.rvalue(c.intType)
			}
			return fmt.Sprintf("make(chan %v %v)", elem.id, cap)
		},
	}
}

func (c *Context) sliceOf(elem *Type) *Type {
	return &Type{
		id:    Id(fmt.Sprintf("[]%v", elem.id)),
		class: ClassSlice,
		ktyp:  elem,
		literal: func() string {
			return fmt.Sprintf("[]%v{}", elem.id)
		}}
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
	case TraitSendable:
		return t.class == ClassChan
	case TraitHashable:
		return t.class != ClassFunction && t.class != ClassMap && t.class != ClassSlice &&
			(t.class != ClassArray || satisfiesTrait(t.ktyp, TraitHashable))
	case TraitPrintable:
		return t.class == ClassBoolean || t.class == ClassNumeric || t.class == ClassString ||
			t.class == ClassPointer || t.class == ClassInterface
	case TraitLenCapable:
		return t.class == ClassString || t.class == ClassSlice || t.class == ClassArray ||
			t.class == ClassMap || t.class == ClassChan
	case TraitFunction:
		return t.class == ClassFunction
	default:
		panic("bad")
	}
}
