// Copyright 2014 Elliott Stoneham and The TARDIS Go Authors
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package haxegoruntime is automatically included in every TARDIS Go transpilation.
// This Go code is used by the Haxe runtime, it must be entirely self-sufficent.
// This Go code is always in the runtime. TODO consider how to slim it down...
package haxegoruntime // long name so as not to clash

import (
	"unicode/utf16"
	"unicode/utf8"
)

// ZiLen is the runtime native string length of the chinese character "字", meaning "written character", which is pronounced "zi" in Mandarin.
// It is initialised by the haxe Go.init() code generated by goclass.go because otherwise the string will be escaped and always be 3 long
var ZiLen int

// UTF16toRunes is a wrapper for utf16.Decode
// TODO review if this can be optimized away
func UTF16toRunes(s []uint16) []rune {
	return utf16.Decode(s)
}

// UTF8toRunes takes a utf8 byte slice and returns the equivalent rune slice
func UTF8toRunes(s []byte) []rune { // TODO rewrite to use sub-slices (now that the system supports them)
	ret := make([]rune, utf8.RuneCount(s))
	si := 0
	for ri := 0; si < len(s) && ri < len(ret); ri++ {
		p := make([]byte, len(s)-si)
		for j := 0; j < (len(s) - si); j++ {
			p[j] = s[si+j]
		}
		aRune, size := utf8.DecodeRune(p)
		ret[ri] = aRune
		si += size
	}
	return ret
}

// Raw2Runes takes the UTF-8 contents of a string and returns the equivalent rune slice
// TODO review if this can be optimized away
func Raw2Runes(s []uint) []rune {
	var tmp = make([]byte, len(s))
	for t := range s {
		tmp[t] = byte(s[t])
	}
	return UTF8toRunes(tmp)
}

// RunesToUTF16 is a wrapper for utf16.Encode
// TODO review if this can be optimized away
func RunesToUTF16(r []rune) []uint16 {
	return utf16.Encode(r)
}

// RunesToUTF8 takes a rune slice and returns the equivalent utf8 byte slice
func RunesToUTF8(r []rune) []byte {
	var ret []byte
	ret = make([]byte, 0)
	for i := range r {
		l := utf8.RuneLen(r[i])
		if l > 0 {
			buf := make([]byte, l)
			utf8.EncodeRune(buf, r[i])
			ret = append(ret, buf...)
		} else {
			buf := make([]byte, utf8.RuneLen(utf8.RuneError))
			utf8.EncodeRune(buf, utf8.RuneError)
			ret = append(ret, buf...)
		}
	}
	return ret
}

// Runes2Raw takes a rune slice and returns a UTF-8 integer slice representing the underlying string
// TODO review if this can be optimized away
func Runes2Raw(r []rune) []uint {
	retUint8 := RunesToUTF8(r)
	var tmpIntS = make([]uint, len(retUint8))
	for tmpI := range retUint8 {
		tmpIntS[tmpI] = uint(retUint8[tmpI])
	}
	return tmpIntS
}

// Rune2Raw takes an individual rune and returns the UTF-8 integer slice representing it
func Rune2Raw(oneRune rune) []uint { // make a string from a single rune
	r := make([]rune, 1)
	r[0] = oneRune
	return Runes2Raw(r)
}