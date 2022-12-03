// package gv implements a compiler for a subset of Go that emits
// SPIR-V.
package gv

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Compiler struct {
	MainPkgPath string

	out *bytes.Buffer

	fileSet *token.FileSet
	types   *types.Info

	externs         map[types.Object]string
	fieldIndices    map[*types.Var]int
	methodRenames   map[types.Object]string
	methodFieldTags map[types.Object]string
	genTypeExprs    map[types.Type]string
	genTypeDefns    map[*ast.TypeSpec]string
	genTypeMetas    map[*ast.TypeSpec]string
	genFuncDecls    map[*ast.FuncDecl]string

	errors *strings.Builder
}

func (c *Compiler) Compile() ([]byte, error) {
	// initialize maps
	c.externs = make(map[types.Object]string)
	c.fieldIndices = make(map[*types.Var]int)
	c.methodRenames = make(map[types.Object]string)
	c.methodFieldTags = make(map[types.Object]string)
	c.genTypeExprs = make(map[types.Type]string)
	c.genTypeDefns = make(map[*ast.TypeSpec]string)
	c.genTypeMetas = make(map[*ast.TypeSpec]string)
	c.genFuncDecls = make(map[*ast.FuncDecl]string)

	// initialize state
	c.errors = new(strings.Builder)
	c.out = new(bytes.Buffer)

	// load main package
	packagesConfig := &packages.Config{
		Mode: packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
	loadPkgs, err := packages.Load(packagesConfig, c.MainPkgPath)
	if err != nil {
		return nil, err
	}
	if len(loadPkgs) == 0 {
		return nil, fmt.Errorf("no packages found")
	}
	pkgLoadErrors := false
	for _, pkg := range loadPkgs {
		for _, err := range pkg.Errors {
			pkgLoadErrors = true
			if err.Pos != "" {
				fmt.Fprintf(c.errors, "%s: %s\n", err.Pos, err.Msg)
			} else {
				fmt.Fprintln(c.errors, err.Msg)
			}
		}
	}
	if pkgLoadErrors {
		return nil, fmt.Errorf("package load errors: %s", c.errors.String())
	}

	// Collect packages
	var pkgs []*packages.Package
	{
		visited := make(map[*packages.Package]bool)
		var visit func(pkg *packages.Package)
		visit = func(pkg *packages.Package) {
			if !visited[pkg] {
				visited[pkg] = true
				for _, dep := range pkg.Imports {
					visit(dep)
				}
				pkgs = append(pkgs, pkg)
				if pkg.Fset != c.fileSet {
					c.err(0, "internal error: filesets differ")
				}
			}
		}
		for _, pkg := range loadPkgs {
			visit(pkg)
		}
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].ID < pkgs[j].ID
	})

	// Collect types info
	c.types = &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Instances:  make(map[*ast.Ident]types.Instance),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	for _, pkg := range pkgs {
		for k, v := range pkg.TypesInfo.Types {
			c.types.Types[k] = v
		}
		for k, v := range pkg.TypesInfo.Instances {
			c.types.Instances[k] = v
		}
		for k, v := range pkg.TypesInfo.Defs {
			c.types.Defs[k] = v
		}
		for k, v := range pkg.TypesInfo.Uses {
			c.types.Uses[k] = v
		}
		for k, v := range pkg.TypesInfo.Implicits {
			c.types.Implicits[k] = v
		}
		for k, v := range pkg.TypesInfo.Selections {
			c.types.Selections[k] = v
		}
		for k, v := range pkg.TypesInfo.Scopes {
			c.types.Scopes[k] = v
		}
	}
	// Collect top-level decls and exports in output order
	var typeSpecs []*ast.TypeSpec
	var valueSpecs []*ast.ValueSpec
	var funcDecls []*ast.FuncDecl
	exports := make(map[types.Object]bool)
	behaviors := make(map[types.Object]bool)
	{
		objTypeSpecs := make(map[types.Object]*ast.TypeSpec)
		objValueSpecs := make(map[types.Object]*ast.ValueSpec)
		for _, pkg := range pkgs {
			for _, file := range pkg.Syntax {
				for _, decl := range file.Decls {
					switch decl := decl.(type) {
					case *ast.GenDecl:
						for _, spec := range decl.Specs {
							switch spec := spec.(type) {
							case *ast.TypeSpec:
								objTypeSpecs[c.types.Defs[spec.Name]] = spec
							case *ast.ValueSpec:
								for _, name := range spec.Names {
									objValueSpecs[c.types.Defs[name]] = spec
								}
							}
						}
					}
				}
			}
		}
		typeSpecVisited := make(map[*ast.TypeSpec]bool)
		valueSpecVisited := make(map[*ast.ValueSpec]bool)
		for _, pkg := range pkgs {
			for _, file := range pkg.Syntax {
				for _, decl := range file.Decls {
					switch decl := decl.(type) {
					case *ast.GenDecl:
						for _, spec := range decl.Specs {
							switch spec := spec.(type) {
							case *ast.TypeSpec:
								var visitTypeSpec func(typeSpec *ast.TypeSpec, export bool)
								visitTypeSpec = func(typeSpec *ast.TypeSpec, export bool) {
									if _, ok := c.externs[c.types.Defs[typeSpec.Name]]; ok {
										return
									}
									obj := c.types.Defs[typeSpec.Name]
									visited := typeSpecVisited[typeSpec]
									if visited && !(export && !exports[obj]) {
										return
									}
									if !visited {
										typeSpecVisited[typeSpec] = true
										if structType, ok := typeSpec.Type.(*ast.StructType); ok {
											for _, field := range structType.Fields.List {
												if field.Names == nil {
													if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "Behavior" {
														behaviors[obj] = true
														export = true
													}
												}
											}
										}
									}
									if export {
										exports[obj] = true
									}
									ast.Inspect(typeSpec.Type, func(node ast.Node) bool {
										if ident, ok := node.(*ast.Ident); ok {
											if typeSpec, ok := objTypeSpecs[c.types.Uses[ident]]; ok {
												visitTypeSpec(typeSpec, export)
											}
										}
										return true
									})
									if !visited {
										typeSpecs = append(typeSpecs, typeSpec)
									}
								}
								visitTypeSpec(spec, false)
							case *ast.ValueSpec:
								var visitValueSpec func(valueSpec *ast.ValueSpec)
								visitValueSpec = func(valueSpec *ast.ValueSpec) {
									if valueSpecVisited[valueSpec] {
										return
									}
									valueSpecVisited[valueSpec] = true
									ast.Inspect(valueSpec, func(node ast.Node) bool {
										if ident, ok := node.(*ast.Ident); ok {
											if valueSpec, ok := objValueSpecs[c.types.Uses[ident]]; ok {
												visitValueSpec(valueSpec)
											}
										}
										return true
									})
									extern := false
									for _, name := range spec.Names {
										if _, ok := c.externs[c.types.Defs[name]]; ok {
											extern = true
										}
									}
									if !extern {
										valueSpecs = append(valueSpecs, valueSpec)
									}
								}
								visitValueSpec(spec)
							}
						}
					case *ast.FuncDecl:
						if _, ok := c.externs[c.types.Defs[decl.Name]]; !ok {
							funcDecls = append(funcDecls, decl)
						}
					}
				}
			}
		}
	}

	// output SPIR-V
	{
		// header
		// types
		for _, typeSpec := range typeSpecs {
			if _, ok := c.genTypeDefns[typeSpec]; ok {
				continue
			}
		}
		// meta ?
		// variables ?
		// function definitions ?
	}

	// load main package
	return nil, nil
}

func (c *Compiler) err(p token.Pos, str string, args ...any) {
	fmt.Fprintf(c.errors, "%s: ", c.fileSet.PositionFor(p, true))
	fmt.Fprintf(c.errors, str, args...)
	fmt.Fprintln(c.errors)
}
