package main

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

type Type struct {
	id      Id
	literal func() string
	class   TypeClass
	ktyp    *Type
	vtyp    *Type
}

func builtinTypes() []*Type {
  return []*Type{
    &Type{id: "bool", class: ClassBoolean, literal: func() string { return "false" }},
    &Type{id: "int", class: ClassNumeric, literal: func() string { return "1" }},
    &Type{id: "int16", class: ClassNumeric, literal: func() string { return "int16(1)" }},
    &Type{id: "float64", class: ClassNumeric, literal: func() string { return "1.1" }},
    &Type{id: "string", class: ClassString, literal: func() string { return "\"foo\"" }},
  }
}