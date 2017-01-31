// This is a copy of the reflect DeepEqual function with a few differences:
//  1. DeepCompare produces a list of differences, useful for dianosing mapping errors
//  2. DeepCompare does not short-circuit to false when a the first difference is found
//  3. DeepCompare does not test unexported fields, because the reflect functions
//     necessary to do this are themselves unexported, and I didn't feel like copying
//     the entire reflect package just to add this one function

// Deep comparison via reflection

package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// During deepValueEqual, must keep track of checks that are
// in progress.  The comparison algorithm assumes that all
// checks in progress are true when it reencounters them.
// Visited comparisons are stored in a map indexed by visit.
type visit struct {
	a1  uintptr
	a2  uintptr
	typ reflect.Type
}

// Tests for deep equality using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueCompare(v1, v2 reflect.Value, visited *map[visit]bool, depth int, diffs *[]string) bool {
	// padding to create nested messages
	p := strings.Repeat("   ", depth)
	if !v1.IsValid() || !v2.IsValid() {
		*diffs = append(*diffs, fmt.Sprintf("%vValidity mismatch: %v, %v", p, v1.IsValid(), v2.IsValid()))
		return v1.IsValid() == v2.IsValid()
	}
	if v1.Type() != v2.Type() {
		*diffs = append(*diffs, fmt.Sprintf("%vType mismatch: %v, %v", p, v1.Type().Name(), v2.Type().Name()))
		return false
	}

	/* DEBUG
	if depth > 10 {
		fmt.Printf("REACHED DEPTH %v, QUITTING", depth)
		*diffs = append(*diffs, "REACHED DEPTH > 10, QUITTING")
		return true
	}
	if len(*diffs) > 100 {
		fmt.Printf("FOUND %v MISMATCHES, QUITTING", len(*diffs))
		*diffs = append(*diffs, "FOUND > 100 MISMATCHES, QUITTING")
		return true
	}
	if len(*diffs) > 10 {
		_ = "breakpoint"
	}*/

	hard := func(k reflect.Kind) bool {
		switch k {
		case reflect.Array, reflect.Map, reflect.Slice, reflect.Struct:
			return true
		}
		return false
	}

	if v1.CanAddr() && v2.CanAddr() && hard(v1.Kind()) {
		addr1 := v1.UnsafeAddr()
		addr2 := v2.UnsafeAddr()
		if addr1 > addr2 {
			// Canonicalize order to reduce number of entries in visited.
			addr1, addr2 = addr2, addr1
		}

		// Short circuit if references are identical ...
		if addr1 == addr2 {
			return true
		}

		// ... or already seen
		typ := v1.Type()
		v := visit{addr1, addr2, typ}
		if (*visited)[v] {
			return true
		}

		// Remember for later.
		(*visited)[v] = true
	}

	switch v1.Kind() {
	case reflect.Array:
		rtn := true
		for i := 0; i < v1.Len(); i++ {
			localDiffs := make([]string, 0)
			thisMatch := deepValueCompare(v1.Index(i), v2.Index(i), visited, depth+1, &localDiffs)
			if !thisMatch {
				rtn = false
				*diffs = append(*diffs, fmt.Sprintf("%vMismatch in element %v", p, i))
				*diffs = append(*diffs, localDiffs...)
			}
		}
		return rtn
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			*diffs = append(*diffs, fmt.Sprintf("%vIsNil mismatch: %v, %v", p, v1.IsNil(), v2.IsNil()))
			return false
		}
		if v1.Len() != v2.Len() {
			*diffs = append(*diffs, fmt.Sprintf("%vLength mismatch: %v, %v", p, v1.Len(), v2.Len()))
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		rtn := true
		for i := 0; i < v1.Len(); i++ {
			localDiffs := make([]string, 0)
			thisMatch := deepValueCompare(v1.Index(i), v2.Index(i), visited, depth+1, &localDiffs)
			if !thisMatch {
				rtn = false
				*diffs = append(*diffs, fmt.Sprintf("%vMismatch in element %v", p, i))
				*diffs = append(*diffs, localDiffs...)
			}
		}
		return rtn
	case reflect.Interface:
		if v1.IsNil() && v2.IsNil() {
			return true
		}
		if v1.IsNil() != v2.IsNil() {
			*diffs = append(*diffs, fmt.Sprintf("%vIsNil mismatch: %v, %v", p, v1.IsNil(), v2.IsNil()))
			return false
		}
		return deepValueCompare(v1.Elem(), v2.Elem(), visited, depth+1, diffs)
	case reflect.Ptr:
		return deepValueCompare(v1.Elem(), v2.Elem(), visited, depth+1, diffs)
	case reflect.Struct:
		rtn := true
		structType := v1.Type()
		for i, n := 0, v1.NumField(); i < n; i++ {
			localDiffs := make([]string, 0)
			f1 := v1.Field(i)
			f2 := v2.Field(i)
			// This condition is what excludes unexported fields
			// without it, some unexported fields are compared, but otherwise
			// will throw panics
			if structType.Field(i).PkgPath == "" {
				thisMatch := deepValueCompare(f1, f2, visited, depth+1, &localDiffs)
				if !thisMatch {
					fieldname := structType.Field(i).Name
					rtn = false
					*diffs = append(*diffs, fmt.Sprintf("%vMismatch in field %v:", p, fieldname))
					*diffs = append(*diffs, localDiffs...)
				}
			}
		}
		return rtn
	case reflect.Map:
		if v1.IsNil() && v2.IsNil() {
			return true
		}
		if v1.IsNil() != v2.IsNil() {
			*diffs = append(*diffs, fmt.Sprintf("%vIsNil mismatch: %v, %v", p, v1.IsNil(), v2.IsNil()))
			return false
		}
		if v1.Len() != v2.Len() {
			*diffs = append(*diffs, fmt.Sprintf("%vLength mismatch: %v, %v", p, v1.Len(), v2.Len()))
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}

		rtn := true
		for kn, k := range v1.MapKeys() {
			localDiffs := make([]string, 0)
			thisMatch := deepValueCompare(v1.MapIndex(k), v2.MapIndex(k), visited, depth+1, &localDiffs)
			if !thisMatch {
				rtn = false
				*diffs = append(*diffs, fmt.Sprintf("%vMismatch in map element %v (%v):", p, kn, k))
				*diffs = append(*diffs, localDiffs...)
			}
		}
		return rtn
	case reflect.Func:
		if v1.IsNil() && v2.IsNil() {
			return true
		}
		// Can't do better than this:
		*diffs = append(*diffs, fmt.Sprintf("%vMismatch in function: %v, %v", p, v1, v2))
		return false
	default:
		// Normal equality suffices
		//DEBUG fmt.Println(v1.Type().Name())
		v1i, v2i := v1.Interface(), v2.Interface()
		if v1i != v2i {
			tn := v1.Type().Name()
			v1s, v2s := fmt.Sprintf("%v", v1i), fmt.Sprintf("%v", v2i)
			if tn == "string" {
				v1s, v2s = fmt.Sprintf("\"%v\"", v1i), fmt.Sprintf("\"%v\"", v2i)
			}
			*diffs = append(*diffs, fmt.Sprintf("%vMismatch in %v values: %v, %v", p, tn, v1s, v2s))
			return false
		}
		return true
	}
}

// DeepCompare is an adaptation of reflect.DeepEqual which tests for deep equality. It uses normal == equality where
// possible but will scan elements of arrays, slices, maps, and fields of
// structs. In maps, keys are compared with == but elements use deep
// equality. DeepEqual correctly handles recursive types. Functions are equal
// only if they are both nil.
// An empty slice is not equal to a nil slice.
// DeepCompare does not short-circuit like DeepEqual, and returns an array of strings describing differences found
func DeepCompare(a1, a2 interface{}) (bool, []string) {
	a1n := (a1 == nil)
	a2n := (a2 == nil)
	if a1n && a2n {
		return true, nil
	}
	if a1n || a2n {
		return false, []string{fmt.Sprintf("Nility mismatch: %v, %v", a1n, a2n)}
	}
	v1 := reflect.ValueOf(a1)
	v2 := reflect.ValueOf(a2)
	if v1.Type() != v2.Type() {
		return false, []string{fmt.Sprintf("Type mismatch: %v, %v", v1, v2)}
	}
	visited := make(map[visit]bool)
	diffs := make([]string, 0)
	match := deepValueCompare(v1, v2, &visited, 0, &diffs)
	return match, diffs
}
