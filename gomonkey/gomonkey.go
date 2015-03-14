package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

var (
	seed = flag.Int64("seed", 0, "random generator seed")
	dir  = flag.String("dir", "tmp", "output dir")
)

func main() {
	flag.Parse()
	rand.Seed(*seed)
	unleashMonkeys()
	path := filepath.Join(*dir, "src", "main")
	os.MkdirAll(path, os.ModePerm)
	w, err := os.Create(filepath.Join(path, "main.go"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output file: %v", err)
		os.Exit(1)
	}
	defer w.Close()
	w.WriteString(srcTempl)

	x := vars[len(vars)-1]
	w.WriteString(f("type XT %v\n", x.typ.id))
	w.WriteString(f("var X = XT(gen())\n"))
	w.WriteString(f("func gen() (%v) {\n", x.typ.id))
	for _, v := range vars {
		w.WriteString(f("  %v := %v\n", v.id, v.val))
		w.WriteString(f("  _ = %v\n", v.id))
		w.WriteString(f("runtime.GC()\n"))
	}
	w.WriteString(f("  return %v\n", x.id))
	w.WriteString(f("}\n"))
}

func f(f string, args ...interface{}) string {
	return fmt.Sprintf(f, args...)
}

func rnd(n int) int {
	return rand.Intn(n)
}

func rndBool() bool {
	return rnd(2) == 0
}

func choice(v ...string) string {
	return v[rnd(len(v))]
}

var srcTempl = `
package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	_ "encoding/xml"
	"fmt"
	"reflect"
	"runtime"
)

func main() {
  println("X:", interface{}(X))
  fmt.Printf("X = %+v\n", X)

	done := make(chan bool)
	for i := 0; i < 4; i++ {
		go func() {
			if !reflect.DeepEqual(X, X) {
				panic("not equal")
			}
			json0, err := json.Marshal(X)
			if err != nil {
				panic(err)
			}
			var x1 XT
			err = json.Unmarshal(json0, &x1)
			if err != nil {
				panic(err)
			}
			json1, err := json.Marshal(x1)
			if err != nil {
				panic(err)
			}
			if bytes.Compare(json0, json1) != 0 {
				panic("not equal")
			}
/*
			xml0, err := xml.Marshal(x1)
			if err != nil {
				panic(err)
			}
			var x2 XT
      xml0 = append(append(append([]byte{}, "<XT>"...), xml0...), "</XT>"...)
			err = xml.Unmarshal(xml0, &x2)
			if err != nil {
				fmt.Printf("xml0: %v\n", string(xml0))
				panic(err)
			}
			xml1, err := xml.Marshal(x2)
			if err != nil {
				panic(err)
			}
			if bytes.Compare(xml0, xml1) != 0 {
				println("x2:", interface{}(x2))
				fmt.Printf("x2 = %+v\n", x2)
				fmt.Printf("old: %v\n", string(xml0))
				fmt.Printf("new: %v\n", string(xml1))
				panic("not equal")
			}
*/
			x2 := x1

			var gob0 bytes.Buffer
			enc0 := gob.NewEncoder(&gob0)
			err = enc0.Encode(x2)
			if err != nil {
				panic(err)
			}
			res0 := gob0.Bytes()
			dec0 := gob.NewDecoder(&gob0)
			var x3 XT
			err = dec0.Decode(&x3)
			if err != nil {
				panic(err)
			}
			var gob1 bytes.Buffer
			enc1 := gob.NewEncoder(&gob1)
			err = enc1.Encode(x3)
			if err != nil {
				panic(err)
			}
			res1 := gob1.Bytes()
			if bytes.Compare(res0, res1) != 0 {
				fmt.Printf("old: %+v\n", res0)
				fmt.Printf("new: %+v\n", res1)
				panic("not equal")
			}

			if !reflect.DeepEqual(X, x3) {
				println("X:", interface{}(X))
				println("x:", interface{}(x3))
				fmt.Printf("X0 = %+v\n", X)
				fmt.Printf("x3 = %+v\n", x3)
				panic("not equal")
			}
			done <- true
		}()
	}
	for i := 0; i < 4; i++ {
		<-done
	}
}

`
