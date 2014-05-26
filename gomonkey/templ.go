// +build

package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
)

func main() {
	done := make(chan bool)
	for i := 0; i < 4; i++ {
		go func() {
			if !reflect.DeepEqual(X, X) {
				println("X:", interface{}(X))
				fmt.Printf("X = %+v\n", X)
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

			xml0, err := xml.Marshal(x1)
			if err != nil {
				panic(err)
			}
			var x2 XT
			err = xml.Unmarshal(xml0, &x2)
			if err != nil {
				panic(err)
			}
			xml1, err := xml.Marshal(x2)
			if err != nil {
				panic(err)
			}
			if bytes.Compare(xml0, xml1) != 0 {
				panic("not equal")
			}

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
				println("x:", interface{}(x1))
				fmt.Printf("X = %+v, x1 = %+v\n", X, x1)
				panic("not equal")
			}
			done <- true
		}()
	}
	for i := 0; i < 4; i++ {
		<-done
	}
}

type XT string

var X = XT("abc")
