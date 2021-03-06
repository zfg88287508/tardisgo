// Copyright 2014 Elliott Stoneham and The TARDIS Go Authors
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package pogo

import (
	"fmt"
	"go/token"
	"sort"
	"strconv"
	"strings"

	"go/constant"
	"golang.org/x/tools/go/ssa"
)

// Recycle the Compilation resources.
func (comp *Compilation) Recycle() { LanguageList[comp.TargetLang] = LanguageEntry{} }

// Compile provides the entry point for the pogo package,
// returning a pogo.Compilation structure and error
func Compile(mainPkg *ssa.Package, debug, trace bool, langName, testFSname string) (*Compilation, error) {
	comp := &Compilation{
		mainPackage: mainPkg,
		rootProgram: mainPkg.Prog,
		DebugFlag:   debug,
		TraceFlag:   trace,
	}

	k, e := FindTargetLang(langName)
	if e != nil {
		return nil, e
	}
	// make a new language list entry for this compilation
	newLang := LanguageList[k]
	languageListAppendMutex.Lock()
	LanguageList = append(LanguageList, newLang)
	comp.TargetLang = len(LanguageList) - 1
	languageListAppendMutex.Unlock()
	LanguageList[comp.TargetLang].Language =
		LanguageList[comp.TargetLang].Language.InitLang(
			comp, &LanguageList[comp.TargetLang])
	LanguageList[comp.TargetLang].TestFS = testFSname
	//fmt.Printf("DEBUG created TargetLang[%d]=%#v\n",
	//	comp.TargetLang, LanguageList[comp.TargetLang])

	comp.initErrors()
	comp.initTypes()
	comp.setupPosHash()
	comp.loadSpecialConsts()
	comp.emitFileStart()
	comp.emitFunctions()
	comp.emitGoClass(comp.mainPackage)
	comp.emitTypeInfo()
	comp.emitFileEnd()
	if comp.hadErrors && comp.stopOnError {
		err := fmt.Errorf("no output files generated")
		comp.LogError("", "pogo", err)
		return nil, err
	}
	comp.writeFiles()
	return comp, nil
}

// The main Go class contains those elements that don't fit in functions
func (comp *Compilation) emitGoClass(mainPkg *ssa.Package) {
	comp.emitGoClassStart()
	comp.emitNamedConstants()
	comp.emitGlobals()
	comp.emitGoClassEnd(mainPkg)
	comp.WriteAsClass("Go", "")
}

// special constant name used in TARDIS Go to put text in the header of files
const pogoHeader = "tardisgoHeader"
const pogoLibList = "tardisgoLibList"

func (comp *Compilation) loadSpecialConsts() {
	hxPkg := ""
	l := comp.TargetLang
	ph := LanguageList[l].HeaderConstVarName
	targetPackage := LanguageList[l].PackageConstVarName
	header := ""
	allPack := comp.rootProgram.AllPackages()
	sort.Sort(PackageSorter(allPack))
	for _, pkg := range allPack {
		allMem := MemberNamesSorted(pkg)
		for _, mName := range allMem {
			mem := pkg.Members[mName]
			if mem.Token() == token.CONST {
				switch mName {
				case ph, pogoHeader: // either the language-specific constant, or the standard one
					lit := mem.(*ssa.NamedConst).Value
					switch lit.Value.Kind() {
					case constant.String:
						h, err := strconv.Unquote(lit.Value.String())
						if err != nil {
							comp.LogError(comp.CodePosition(lit.Pos())+"Special pogo header constant "+ph+" or "+pogoHeader,
								"pogo", err)
						} else {
							header += h + "\n"
						}
					}
				case targetPackage:
					lit := mem.(*ssa.NamedConst).Value
					switch lit.Value.Kind() {
					case constant.String:
						hp, err := strconv.Unquote(lit.Value.String())
						if err != nil {
							comp.LogError(comp.CodePosition(lit.Pos())+"Special targetPackage constant ", "pogo", err)
						}
						hxPkg = hp
					default:
						comp.LogError(comp.CodePosition(lit.Pos()), "pogo",
							fmt.Errorf("special targetPackage constant not a string"))
					}
				case pogoLibList:
					lit := mem.(*ssa.NamedConst).Value
					switch lit.Value.Kind() {
					case constant.String:
						lrp, err := strconv.Unquote(lit.Value.String())
						if err != nil {
							comp.LogError(comp.CodePosition(lit.Pos())+"Special "+pogoLibList+" constant ", "pogo", err)
						}
						comp.LibListNoDCE = strings.Split(lrp, ",")
						for lib := range comp.LibListNoDCE {
							comp.LibListNoDCE[lib] = strings.TrimSpace(comp.LibListNoDCE[lib])
						}
					default:
						comp.LogError(comp.CodePosition(lit.Pos()), "pogo",
							fmt.Errorf("special targetPackage constant not a string"))
					}
				}
			}
		}
	}
	comp.hxPkgName = hxPkg
	comp.headerText = header
}

// emit the standard file header for target language
func (comp *Compilation) emitFileStart() {
	l := comp.TargetLang
	fmt.Fprintln(&LanguageList[l].buffer,
		LanguageList[l].FileStart(comp.hxPkgName, comp.headerText))
}

// emit the tail of the required language file
func (comp *Compilation) emitFileEnd() {
	l := comp.TargetLang
	fmt.Fprintln(&LanguageList[l].buffer, LanguageList[l].FileEnd())
	for w := range comp.warnings {
		comp.emitComment(comp.warnings[w])
	}
	comp.emitComment("Package List:")
	allPack := comp.rootProgram.AllPackages()
	sort.Sort(PackageSorter(allPack))
	for pkgIdx := range allPack {
		comp.emitComment(" " + allPack[pkgIdx].String())
	}
}

// emit the start of the top level type definition for each language
func (comp *Compilation) emitGoClassStart() {
	l := comp.TargetLang
	fmt.Fprintln(&LanguageList[l].buffer, LanguageList[l].GoClassStart())
}

// emit the end of the top level type definition for each language file
func (comp *Compilation) emitGoClassEnd(pak *ssa.Package) {
	l := comp.TargetLang
	fmt.Fprintln(&LanguageList[l].buffer, LanguageList[l].GoClassEnd(pak))
}

/*
func (comp *Compilation) UsingPackage(pkgName string) bool {
	//println("DEBUG UsingPackage() looking for: ", pkgName)
	pkgName = "package " + pkgName
	pkgs := comp.rootProgram.AllPackages()
	for p := range pkgs {
		//println("DEBUG UsingPackage() considering pkg: ", pkgs[p].String())
		if pkgs[p].String() == pkgName {
			//println("DEBUG UsingPackage()  ", pkgName, " = true")
			return true
		}
	}
	//println("DEBUG UsingPackage()  ", pkgName, " =false")
	return false
}
*/
