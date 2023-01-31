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
type TypeLookupFn func(packagePath, packageIdentifier, typeName string) (reflect.Type, bool)

func (i TypesIndex) Lookup(_, packageIdentifier, typeName string) (reflect.Type, bool) {
	aKey := typeName
	if packageIdentifier != "" {
		aKey = packageIdentifier + "." + typeName
	}

	rType, ok := i[aKey]
	return rType, ok
}

func ParseTypes(path string) (*DirTypes, error) {
	fileSet := token.NewFileSet()
	packageFiles, err := parser.ParseDir(fileSet, path, nil, 0)
	if err != nil {
		return nil, err
	}

	return indexPackageMetadata(packageFiles, path)
}

func indexPackageMetadata(packages map[string]*ast.Package, path string) (*DirTypes, error) {
	types := NewDirTypes(path)
	for _, aPackage := range packages {
		if err := indexPackage(types, aPackage); err != nil {
			return nil, err
		}
	}

	return types, nil
}

func indexPackage(types *DirTypes, aPackage *ast.Package) error {
	for path, file := range aPackage.Files {
		types.addScope(path, file.Scope)
		if err := types.addImports(path, file); err != nil {
			return err
		}

		for _, decl := range file.Decls {
			indexFunc(types, decl)

			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				indexTypeSpec(types, path, spec)
			}
		}
	}

	return nil
}

func indexFunc(types *DirTypes, spec interface{}) {
	funcSpec, ok := asFuncDecl(spec)
	if !ok {
		return
	}

	for _, field := range funcSpec.Recv.List {
		receiverType, ok := derefIdentIfNeeded(field.Type)
		if ok {
			types.registerMethod(receiverType.Name, funcSpec)
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

func indexTypeSpec(types *DirTypes, path string, spec ast.Spec) {
	typeSpec, ok := asTypeSpec(spec)
	if !ok {
		return
	}

	types.indexTypeSpec(path, typeSpec)
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

	rType, err := matchType(expr, shouldUnquote, lookup)
	if err != nil {
		return nil, err
	}

	return rType, nil
}

func matchType(expr ast.Node, shouldUnquote bool, lookup TypeLookupFn) (reflect.Type, error) {
	switch actual := expr.(type) {
	case *ast.StarExpr:
		rType, err := matchType(actual.X, shouldUnquote, lookup)
		if err != nil {
			return nil, err
		}

		return reflect.PtrTo(rType), nil
	case *ast.StructType:
		rFields := make([]reflect.StructField, 0, len(actual.Fields.List))

		for _, field := range actual.Fields.List {
			tag := ""
			if field.Tag != nil {
				unquote, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					return nil, err
				}

				tag = unquote
			}

			fieldType, err := matchType(field.Type, shouldUnquote, lookup)
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

			typeName := packageIdent.Name + "." + actual.Sel.Name
			rType, ok := lookup("", packageIdent.Name, actual.Sel.Name)
			if !ok {
				return nil, typeNotFoundError(typeName)
			}
			return rType, nil
		} else {
			rType, ok := lookup("", "", actual.Sel.Name)
			if !ok {
				return nil, typeNotFoundError(actual.Sel.Name)
			}
			return rType, nil
		}

	case *ast.ArrayType:
		rType, err := matchType(actual.Elt, shouldUnquote, lookup)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(rType), nil
	case *ast.MapType:
		keyType, err := matchType(actual.Key, shouldUnquote, lookup)
		if err != nil {
			return nil, err
		}

		valueType, err := matchType(actual.Value, shouldUnquote, lookup)
		if err != nil {
			return nil, err
		}

		return reflect.MapOf(keyType, valueType), nil
	case *ast.InterfaceType:
		return InterfaceType, nil
	case *ast.TypeSpec:
		return matchType(actual.Type, shouldUnquote, lookup)
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
			rType, ok := lookup("", "", actual.Name)
			if !ok {
				return nil, typeNotFoundError(actual.Name)
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
