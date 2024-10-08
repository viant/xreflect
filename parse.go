package xreflect

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/mod/modfile"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
)

func ParseTypes(path string, options ...Option) (*DirTypes, error) {
	dirTypes := NewDirTypes(path)
	dirTypes.options.Apply(options...)
	fileSet := token.NewFileSet()
	packageFiles, err := parser.ParseDir(fileSet, path, nil, dirTypes.options.parseMode)
	if err != nil {
		return nil, err
	}
	if err = dirTypes.indexPackages(packageFiles); err != nil {
		return nil, err
	}

	dirTypes.ModulePath = detectModulePath(path)
	return dirTypes, nil
}

func detectModulePath(aPath string) string {
	parts := strings.Split(aPath, "/")
	var index int
	var aFile *modfile.File
	for i := len(parts) - 1; i >= 0; i-- {
		aPath = strings.Join(parts[:i], "/")
		if isFileExists(path.Join(aPath, "go.mod")) {
			index = i
			data, _ := os.ReadFile(path.Join(aPath, "go.mod"))
			aFile, _ = modfile.Parse("", data, nil)
			break
		}
	}
	if aFile == nil || aFile.Module == nil {
		return ""
	}
	return path.Join(aFile.Module.Mod.Path, strings.Join(parts[index:], "/"))
}

func (t *DirTypes) indexPackages(packages map[string]*ast.Package) error {
	for _, aPackage := range packages {
		if err := t.indexPackage(aPackage); err != nil {
			return err
		}
	}
	return nil
}

func (t *DirTypes) indexPackage(aPackage *ast.Package) error {
	for path, file := range aPackage.Files {
		t.addPackage(path, aPackage.Name)
		t.addScope(path, file.Scope)
		if err := t.addImports(path, file); err != nil {
			return err
		}
		for _, decl := range file.Decls {
			t.indexFunc(decl)
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				t.indexTypeSpec(path, aPackage.Name, spec)
			}
		}
	}

	return nil
}

func (t *DirTypes) indexFunc(spec interface{}) {
	funcSpec, ok := asFuncDecl(spec)
	if !ok {
		return
	}

	recv := funcSpec.Recv
	if recv == nil {
		return
	}

	for _, field := range recv.List {
		receiverType, ok := derefIdentIfNeeded(field.Type)
		if ok {
			t.registerMethod(receiverType.Name, funcSpec)
		}
	}
}

func derefIdentIfNeeded(expr ast.Expr) (*ast.Ident, bool) {
	ident, ok := asIdent(expr)
	if ok {
		return ident, ok
	}

	starExpr, ok := expr.(*ast.StarExpr)
	if ok {
		return derefIdentIfNeeded(starExpr.X)
	}
	return nil, false
}

func asFuncDecl(spec interface{}) (*ast.FuncDecl, bool) {
	decl, ok := spec.(*ast.FuncDecl)
	return decl, ok
}

func (t *DirTypes) indexTypeSpec(path string, pkg string, spec ast.Spec) {
	typeSpec, ok := asTypeSpec(spec)
	if !ok {
		return
	}
	t.registerTypeSpec(path, pkg, typeSpec)
}

func Parse(dataType string, opts ...Option) (reflect.Type, error) {
	o := options{}
	o.Apply(opts...)
	lookup := o.lookup
	if lookup == nil && o.Registry != nil {
		lookup = o.Registry.Lookup
	}
	var registry *Types
	if lookup == nil {
		registry = NewTypes(opts...)
		lookup = registry.Lookup
	}
	expr, err := parser.ParseExpr(dataType)
	if err != nil {
		return nil, err
	}
	types := NewDirTypes("")
	types.Apply(WithTypeLookup(lookup), WithPackage(o.Package), WithRegistry(o.Registry), WithModule(o.module, o.moduleLocation))
	typeSpec := &TypeSpec{DirTypes: types}
	pkgPath := ""
	rType, err := typeSpec.matchType(types.Package, &pkgPath, nil, expr, o.GoImports)
	if err != nil {
		return nil, err
	}
	return rType, nil
}

func (t *TypeSpec) matchType(pkg string, pkgPath *string, spec *ast.TypeSpec, expr ast.Node, imps GoImports) (reflect.Type, error) {
	if len(imps) > 0 {
		t.options.GoImports = imps
	} else {
		imps = t.options.GoImports
	}
	switch actual := expr.(type) {
	case *ast.StarExpr:
		rType, err := t.matchType(pkg, pkgPath, spec, actual.X, imps)
		if err != nil {
			return nil, err
		}
		return reflect.PtrTo(rType), nil
	case *ast.StructType:
		if t.options.onStruct != nil {
			t.options.onStruct(spec, actual, nil)
		}
		imps = t.DirTypes.imports[t.path]
		if len(imps) == 0 {
			imps = t.options.GoImports
		}
		rFields := make([]reflect.StructField, 0, len(actual.Fields.List))
		for _, field := range actual.Fields.List {

			if t.onField != nil {
				if err := t.onField(spec.Name.Name, field, imps); err != nil {
					return nil, err
				}
			}
			prevTypeName := ""
			tag := ""
			if field.Tag != nil {
				unquote, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					return nil, err
				}
				tag = unquote
				tag, prevTypeName = RemoveTag(tag, TagTypeName)
			}

			fieldType, err := t.matchType(pkg, pkgPath, spec, field.Type, imps)
			if err != nil {
				return nil, err
			}
			n := Node{Node: field.Type}

			typeName, _ := n.Stringify()
			if prevTypeName != "" {
				if typeName == "" {
					typeName = prevTypeName
				}
				tag += " " + TagTypeName + `:"` + componentType(typeName) + `"`
			}

			for _, name := range field.Names {
				structField := reflect.StructField{
					Name:    name.Name,
					Tag:     reflect.StructTag(tag),
					Type:    fieldType,
					PkgPath: PkgPath(name.Name, pkg),
				}
				structField.Anonymous = name.Name == fieldType.Name() && strings.Contains(string(structField.Tag), "anonymous")
				rFields = append(rFields, structField)
			}
			if len(field.Names) == 0 {
				name := fieldType.Name()
				if name == "" {
					aNode := Node{field.Type}
					name, _ = aNode.Stringify()
					name = rawName(name)
				}
				structField := reflect.StructField{
					Name:      name,
					Tag:       reflect.StructTag(tag),
					Type:      fieldType,
					PkgPath:   PkgPath(name, pkg),
					Anonymous: true,
				}
				rFields = append(rFields, structField)
			}
		}
		return reflect.StructOf(rFields), nil

	case *ast.SelectorExpr:
		packageIdent, ok := asIdent(actual.X)
		if ok {
			r, done := t.tryResolveStandardTypes(packageIdent, actual)
			if done {
				return r, nil
			}
			rType, err := t.lookup("", packageIdent.Name, actual.Sel.Name)
			if err != nil {
				if pkgPath := imps.OwnertPkgPath(packageIdent.Name); pkgPath != "" {
					rType, err = t.lookup("", pkgPath, actual.Sel.Name)
				}
				if err != nil {
					return nil, err
				}
			}
			return rType, nil
		} else {
			rType, err := t.lookup("", "", actual.Sel.Name)
			if err != nil {
				return nil, err
			}
			return rType, nil
		}

	case *ast.ArrayType:
		rType, err := t.matchType(pkg, pkgPath, spec, actual.Elt, imps)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(rType), nil
	case *ast.MapType:
		keyType, err := t.matchType(pkg, pkgPath, spec, actual.Key, imps)
		if err != nil {
			return nil, err
		}
		valueType, err := t.matchType(pkg, pkgPath, spec, actual.Value, imps)
		if err != nil {
			return nil, err
		}
		return reflect.MapOf(keyType, valueType), nil
	case *ast.InterfaceType:
		return InterfaceType, nil
	case *ast.TypeSpec:
		return t.matchType(pkg, pkgPath, actual, actual.Type, imps)
	case *ast.Ident:
		switch actual.Name {
		case "int":
			return IntType, nil
		case "int8":
			return Int8Type, nil
		case "int16":
			return Int16Type, nil
		case "int32":
			return Int32Type, nil
		case "int64":
			return Int64Type, nil
		case "uint":
			return UintType, nil
		case "uint8":
			return Uint8Type, nil
		case "uint16":
			return Uint16Type, nil
		case "uint32":
			return Uint32Type, nil
		case "uint64":
			return Uint64Type, nil
		case "float32":
			return Float32Type, nil
		case "float64":
			return Float64Type, nil
		case "time.Time":
			return TimeType, nil
		case "string":
			return StringType, nil
		case "bool":
			return BoolType, nil
		case "interface":
			return InterfaceType, nil
		default:

			//first lookup within the same package after that fallback to global check
			if rType, err := t.lookup("", pkg, actual.Name); rType != nil {
				return rType, err
			}
			rType, err := t.lookup("", "", actual.Name)
			if err != nil {
				return nil, err
			}
			return rType, nil
		}

	}

	return nil, fmt.Errorf("unsupported %T, %v", expr, expr)
}

var JSONRawMessageType = reflect.TypeOf(json.RawMessage{})

func (t *TypeSpec) tryResolveStandardTypes(packageIdent *ast.Ident, actual *ast.SelectorExpr) (reflect.Type, bool) {
	switch packageIdent.Name {
	case "time":
		switch actual.Sel.Name {
		case "Time":
			return TimeType, true
		}
	case "json":
		switch actual.Sel.Name {
		case "RawMessage":
			return JSONRawMessageType, true
		}
	}
	return nil, false
}

func sourceLocation(t *TypeSpec, imp *GoImport) (string, string) {
	module := t.options.module
	if module == nil {
		return "", ""
	}
	folder := strings.Replace(imp.Module, module.Mod.Path, "", 1)
	location := path.Join(t.options.moduleLocation, folder)
	return location, folder
}

func PkgPath(fieldName string, pkgPath string) (fieldPath string) {
	if fieldName != "" && (fieldName[0] > 'Z' || fieldName[0] < 'A') {
		if pkgPath == "" {
			pkgPath = "autogen"
		}
		fieldPath = pkgPath
	}
	return fieldPath
}

func asTypeSpec(spec ast.Spec) (*ast.TypeSpec, bool) {
	result, ok := spec.(*ast.TypeSpec)
	return result, ok
}

func asIdent(x ast.Expr) (*ast.Ident, bool) {
	ident, ok := x.(*ast.Ident)
	return ident, ok
}

func isFileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return false
	}
	return true
}
