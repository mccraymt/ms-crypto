// Copied with small adaptations from the reflect package in the
// Go source tree. We use testing rather than gocheck to preserve
// as much source equivalence as possible.

// TODO tests for error messages

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Basic struct {
	X int
	Y float32
	z int
}

type NotBasic Basic

type DeepCompareTest struct {
	a, b interface{}
	eq   bool
}

// Simple functions for DeepCompare tests.
var (
	fn1 func()             // nil.
	fn2 func()             // nil.
	fn3 = func() { fn1() } // Not nil.
)

var DeepCompareTests = []DeepCompareTest{
	// Equalities
	{nil, nil, true},
	{1, 1, true},
	{int32(1), int32(1), true},
	{0.5, 0.5, true},
	{float32(0.5), float32(0.5), true},
	{"hello", "hello", true},
	{make([]int, 10), make([]int, 10), true},
	{&[3]int{1, 2, 3}, &[3]int{1, 2, 3}, true},
	{Basic{1, 0.5, 0}, Basic{1, 0.5, 0}, true},
	// unexported values are NOT compared
	{Basic{1, 0.5, 0}, Basic{1, 0.5, 1}, true},
	{error(nil), error(nil), true},
	{map[int]string{1: "one", 2: "two"}, map[int]string{2: "two", 1: "one"}, true},
	{fn1, fn2, true},

	// Inequalities
	{1, 2, false},
	{int32(1), int32(2), false},
	{0.5, 0.6, false},
	{float32(0.5), float32(0.6), false},
	{"hello", "hey", false},
	{make([]int, 10), make([]int, 11), false},
	{&[3]int{1, 2, 3}, &[3]int{1, 2, 4}, false},
	{Basic{1, 0.5, 0}, Basic{1, 0.6, 0}, false},
	{Basic{1, 0, 0}, Basic{2, 0, 0}, false},
	{map[int]string{1: "one", 3: "two"}, map[int]string{2: "two", 1: "one"}, false},
	{map[int]string{1: "one", 2: "txo"}, map[int]string{2: "two", 1: "one"}, false},
	{map[int]string{1: "one"}, map[int]string{2: "two", 1: "one"}, false},
	{map[int]string{2: "two", 1: "one"}, map[int]string{1: "one"}, false},
	{nil, 1, false},
	{1, nil, false},
	{fn1, fn3, false},
	{fn3, fn3, false},

	// Nil vs empty: they're different
	{[]int{}, []int(nil), false},
	{[]int{}, []int{}, true},
	{[]int(nil), []int(nil), true},

	// Mismatched types
	{1, 1.0, false},
	{int32(1), int64(1), false},
	{0.5, "hello", false},
	{[]int{1, 2, 3}, [3]int{1, 2, 3}, false},
	{&[3]interface{}{1, 2, 4}, &[3]interface{}{1, 2, "s"}, false},
	{Basic{1, 0.5, 0}, NotBasic{1, 0.5, 0}, false},
	{map[uint]string{1: "one", 2: "two"}, map[int]string{2: "two", 1: "one"}, false},
}

func TestDeepCompare() bool {
	rtn := true
	idx := 1
	for _, test := range DeepCompareTests {
		//fmt.Printf("%v \n", idx)
		idx++
		r, _ := DeepCompare(test.a, test.b)
		if r != test.eq {
			rtn = false
			fmt.Printf("DeepCompare(%v, %v) = %v, want %v \n", test.a, test.b, r, test.eq)
		}
	}
	return rtn
}

type Recursive struct {
	x int
	r *Recursive
}

func TestDeepCompareRecursiveStruct() bool {
	a, b := new(Recursive), new(Recursive)
	*a = Recursive{12, a}
	*b = Recursive{12, b}
	if eq, _ := DeepCompare(a, b); !eq {
		fmt.Println("DeepCompare(recursive same) = false, want true")
		return false
	}
	return true
}

type _Complex struct {
	A int
	B [3]*_Complex
	C *string
	D map[float64]float64
}

func TestDeepCompareComplexStruct() bool {
	m := make(map[float64]float64)
	stra, strb := "hello", "hello"
	a, b := new(_Complex), new(_Complex)
	*a = _Complex{5, [3]*_Complex{a, b, a}, &stra, m}
	*b = _Complex{5, [3]*_Complex{b, a, a}, &strb, m}
	if eq, _ := DeepCompare(a, b); !eq {
		fmt.Println("DeepCompare(complex same) = false, want true")
		return false
	}
	return true
}

func TestDeepCompareComplexStructInequality() bool {
	m := make(map[float64]float64)
	stra, strb := "hello", "helloo" // Difference is here
	a, b := new(_Complex), new(_Complex)
	*a = _Complex{5, [3]*_Complex{a, b, a}, &stra, m}
	*b = _Complex{5, [3]*_Complex{b, a, a}, &strb, m}
	if eq, _ := DeepCompare(a, b); eq {
		fmt.Println("DeepCompare(complex different) = true, want false")
		return false
	}
	return true
}

type UnexpT struct {
	m map[int]int
}

func TestDeepCompareUnexportedMap(t *testing.T) bool {
	// Check that DeepCompare can look at unexported fields.
	rtn := true
	x1 := UnexpT{map[int]int{1: 2}}
	x2 := UnexpT{map[int]int{1: 2}}
	if eq, _ := DeepCompare(&x1, &x2); !eq {
		fmt.Println("DeepCompare(x1, x2) = false, want true")
		rtn = false
	}

	y1 := UnexpT{map[int]int{2: 3}}
	if eq, _ := DeepCompare(&x1, &y1); eq {
		fmt.Println("DeepCompare(x1, y1) = true, want false")
		rtn = false
	}
	return rtn
}

var _ = Describe("DeepCompare", func() {
	It("should correctly test equality of simple values and structs", func() {
		foo := TestDeepCompare()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly test deep structs", func() {
		foo := TestDeepCompareComplexStruct()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly test complex structs", func() {
		foo := TestDeepCompareComplexStruct()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly detect unequal complex structs", func() {
		foo := TestDeepCompareComplexStructInequality()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

})
