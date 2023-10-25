package xreflect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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
	return dirTypes, nil
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
	types.Apply(WithTypeLookup(lookup), WithPackage(o.Package), WithRegistry(o.Registry), WithPackage(o.Package))
	rType, err := types.matchType(types.Package, nil, expr)
	if err != nil {
		return nil, err
	}
	return rType, nil
}

func (t *DirTypes) matchType(pkg string, spec *ast.TypeSpec, expr ast.Node) (reflect.Type, error) {
	switch actual := expr.(type) {
	case *ast.StarExpr:
		rType, err := t.matchType(pkg, spec, actual.X)
		if err != nil {
			return nil, err
		}
		return reflect.PtrTo(rType), nil
	case *ast.StructType:

		rFields := make([]reflect.StructField, 0, len(actual.Fields.List))
		for _, field := range actual.Fields.List {

			if t.onField != nil {
				if err := t.onField(spec.Name.Name, field); err != nil {
					return nil, err
				}
			}
			tag := ""
			if field.Tag != nil {
				unquote, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					return nil, err
				}
				tag = unquote
			}
			fieldType, err := t.matchType(pkg, spec, field.Type)
			if err != nil {
				return nil, err
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
			switch packageIdent.Name {
			case "time":
				switch actual.Sel.Name {
				case "Time":
					return TimeType, nil
				}
			}

			rType, err := t.lookup("", packageIdent.Name, actual.Sel.Name)
			if err != nil {
				return nil, err
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
		rType, err := t.matchType(pkg, spec, actual.Elt)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(rType), nil
	case *ast.MapType:
		keyType, err := t.matchType(pkg, spec, actual.Key)
		if err != nil {
			return nil, err
		}
		valueType, err := t.matchType(pkg, spec, actual.Value)
		if err != nil {
			return nil, err
		}
		return reflect.MapOf(keyType, valueType), nil
	case *ast.InterfaceType:
		return InterfaceType, nil
	case *ast.TypeSpec:
		return t.matchType(pkg, actual, actual.Type)
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

func PkgPath(fieldName string, pkgPath string) (fieldPath string) {
	if fieldName != "" && (fieldName[0] > 'Z' || fieldName[0] < 'A') {
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
