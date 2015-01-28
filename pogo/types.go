// Copyright 2014 Elliott Stoneham and The TARDIS Go Authors
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package pogo

import (
	"fmt"
	"reflect"

	"golang.org/x/tools/go/types"
	"golang.org/x/tools/go/types/typeutil"
)

// IsValidInPogo exists to screen out any types that the system does not handle correctly.
// Currently it should say everything is valid.
// TODO review if still required in this form.
func IsValidInPogo(et types.Type, posStr string) bool {
	switch et.(type) {
	case *types.Basic:
		switch et.(*types.Basic).Kind() {
		case types.Bool, types.String, types.Float64, types.Float32,
			types.Int, types.Int8, types.Int16, types.Int32,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Int64, types.Uint64,
			types.Complex64, types.Complex128,
			types.Uintptr, types.UnsafePointer:
			return true
		case types.UntypedInt, types.UntypedRune, types.UntypedBool,
			types.UntypedString, types.UntypedFloat, types.UntypedComplex:
			return true
		default:
			if et.(*types.Basic).String() == "invalid type" { // the type of unused map value itterators!
				return true
			}
			LogError(posStr, "pogo", fmt.Errorf("basic type %s is not supported", et.(*types.Basic).String()))
		}
	case *types.Interface, *types.Slice, *types.Struct, *types.Tuple, *types.Map, *types.Pointer, *types.Array,
		*types.Named, *types.Signature, *types.Chan:
		return true
	default:
		rTyp := reflect.TypeOf(et).String()
		if rTyp == "*ssa.opaqueType" { // the type of map itterators!
			return true
		}
		LogError(posStr, "pogo", fmt.Errorf("type %s is not supported", rTyp))
	}
	return false
}

// TypesEncountered keeps track of the types we encounter using the excellent go.tools/go/types/typesmap package.
var TypesEncountered typeutil.Map

// NextTypeID is used to give each type we come across its own ID.
var NextTypeID = 1 // entry zero is invalid

// LogTypeUse : As the code generator encounters new types it logs them here, returning a string of the ID for insertion into the code.
func LogTypeUse(t types.Type) string {
	r := TypesEncountered.At(t)
	if r != nil {
		return fmt.Sprintf("%d", r)
	}
	TypesEncountered.Set(t, NextTypeID)
	r = NextTypeID
	NextTypeID++
	return fmt.Sprintf("%d", r)
}

// TypesWithMethodSets in a utility function to only return seen types
func TypesWithMethodSets() (sets []types.Type) {
	typs := rootProgram.RuntimeTypes()
	for _, t := range typs {
		if TypesEncountered.At(t) != nil {
			sets = append(sets, t)
		}
	}
	return sets
}

func catchReferencedTypes(et types.Type) {
	LogTypeUse(et)
	//LogTypeUse(types.NewPointer(et))
	switch et.(type) {
	case *types.Named:
		catchReferencedTypes(et.Underlying())
	case *types.Array:
		catchReferencedTypes(et.(*types.Array).Elem())
		//catchReferencedTypes(types.NewSlice(et.(*types.Array).Elem()))
	case *types.Pointer:
		catchReferencedTypes(et.(*types.Pointer).Elem())
	case *types.Slice:
		catchReferencedTypes(et.(*types.Slice).Elem())
	}
}

// Wrapper for target language emitTypeInfo()
func emitTypeInfo() {

	// belt-and-braces could be used here to make sure we capture every type, needed for reflect
	// but this makes the generated code too large for Java & C++
	/*
		for _, pkg := range rootProgram.AllPackages() {
			for _, mem := range pkg.Members {
				catchReferencedTypes(mem.Type())
			}
		}
	*/

	// ...so just get the full info on the types we've seen
	for _, t := range TypesEncountered.Keys() {
		catchReferencedTypes(t)
	}

	l := TargetLang
	fmt.Fprintln(&LanguageList[l].buffer, LanguageList[l].EmitTypeInfo())
}
