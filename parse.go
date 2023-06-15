package xreflect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
)

type TypesIndex map[string]reflect.Type
type TypeLookupFn func(packagePath, packageIdentifier, typeName string) (reflect.Type, error)

func (i TypesIndex) Lookup(_, packageIdentifier, typeName string) (reflect.Type, error) {
	aKey := typeName
	if packageIdentifier != "" {
		aKey = packageIdentifier + "." + typeName
	}

	rType, ok := i[aKey]
	if !ok {
		return nil, fmt.Errorf("not found type %v", aKey)
	}

	return rType, nil
}

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
				t.indexTypeSpec(path, spec)
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

func (t *DirTypes) indexTypeSpec(path string, spec ast.Spec) {
	typeSpec, ok := asTypeSpec(spec)
	if !ok {
		return
	}
	t.registerTypeSpec(path, typeSpec)
}

func Parse(dataType string, extraTypes ...reflect.Type) (reflect.Type, error) {
	return parseWithTypes(dataType, extraTypes, true)
}

func ParseUnquoted(dataType string, extraTypes ...reflect.Type) (reflect.Type, error) {
	return parseWithTypes(dataType, extraTypes, false)
}

func ParseWithLookup(dataType string, shouldUnquote bool, lookup TypeLookupFn) (reflect.Type, error) {
	return parseWithLookup(dataType, shouldUnquote, lookup)
}

func parseWithTypes(dataType string, extraTypes []reflect.Type, shouldUnquote bool) (reflect.Type, error) {
	typesIndex := TypesIndex{}
	for i, extraType := range extraTypes {
		typesIndex[extraType.String()] = extraTypes[i]
	}

	return parseWithLookup(dataType, shouldUnquote, typesIndex.Lookup)
}

func parseWithLookup(dataType string, shouldUnquote bool, lookup TypeLookupFn) (reflect.Type, error) {
	expr, err := parser.ParseExpr(dataType)
	if err != nil {
		return nil, err
	}
	types := NewDirTypes("")
	types.Apply(WithTypeLookupFn(lookup))
	rType, err := types.matchType(expr, shouldUnquote)
	if err != nil {
		return nil, err
	}

	return rType, nil
}

func (t *DirTypes) matchType(expr ast.Node, shouldUnquote bool) (reflect.Type, error) {
	switch actual := expr.(type) {
	case *ast.StarExpr:
		rType, err := t.matchType(actual.X, shouldUnquote)
		if err != nil {
			return nil, err
		}
		return reflect.PtrTo(rType), nil
	case *ast.StructType:
		rFields := make([]reflect.StructField, 0, len(actual.Fields.List))
		for _, field := range actual.Fields.List {
			if t.onField != nil {
				t.onField(field)
			}
			tag := ""
			if field.Tag != nil {
				unquote, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					return nil, err
				}

				tag = unquote
			}
			fieldType, err := t.matchType(field.Type, shouldUnquote)
			if err != nil {
				return nil, err
			}
			for _, name := range field.Names {
				rFields = append(rFields, reflect.StructField{
					Name: name.Name,
					Tag:  reflect.StructTag(tag),
					Type: fieldType,
				})
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
		rType, err := t.matchType(actual.Elt, shouldUnquote)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(rType), nil
	case *ast.MapType:
		keyType, err := t.matchType(actual.Key, shouldUnquote)
		if err != nil {
			return nil, err
		}
		valueType, err := t.matchType(actual.Value, shouldUnquote)
		if err != nil {
			return nil, err
		}
		return reflect.MapOf(keyType, valueType), nil
	case *ast.InterfaceType:
		return InterfaceType, nil
	case *ast.TypeSpec:
		return t.matchType(actual.Type, shouldUnquote)
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
			rType, err := t.lookup("", "", actual.Name)
			if err != nil {
				return nil, err
			}
			return rType, nil
		}
	}

	return nil, fmt.Errorf("unsupported %T, %v", expr, expr)
}

func typeNotFoundError(name string) error {
	return fmt.Errorf("not found type %v", name)
}

func asTypeSpec(spec ast.Spec) (*ast.TypeSpec, bool) {
	result, ok := spec.(*ast.TypeSpec)
	return result, ok
}

func asIdent(x ast.Expr) (*ast.Ident, bool) {
	ident, ok := x.(*ast.Ident)
	return ident, ok
}
