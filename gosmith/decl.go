package main

import (
  _ "math/rand"
)

func (c *Context) declVar() bool {
/*
  n := rand.Intn(4) + 1
  vars := make([]*Var, n)
  for i := 0; i < n; i++ {
    vars[i] = &Var{id: c.newId(), typ: c.existingType()}
  }
  c.F("var %v", vars[0].id)
  for i := 1; i < n; i++ {
    c.F(", %v", vars[i].id)
  }
  c.F(" = ")


	id := c.newId()
	typ := c.existingType()
	switch rand.Intn(2) {
	case 0: // short form
		c.F("%v := ", id)
	case 1: // full form
		c.F("var %v %v = ", id, typ.id)
	}
	c.expression(typ)
	c.F("\n")
  if id != "_" {
    c.vars = append(c.vars, &Var{id: id, typ: typ})
  }
  */
	return true
}