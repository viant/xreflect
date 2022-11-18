package xreflect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"reflect"
	"strconv"
)

type (
	TypeDef struct {
		Name string
		Type reflect.Type
	}

	Modifier func(p reflect.Type) reflect.Type
)

func Parse(dataType string, extraTypes ...reflect.Type) (reflect.Type, error) {
	return parse(dataType, extraTypes, true)
}

func ParseUnquoted(dataType string, extraTypes ...reflect.Type) (reflect.Type, error) {
	return parse(dataType, extraTypes, false)
}

func parse(dataType string, extraTypes []reflect.Type, shouldUnquote bool) (reflect.Type, error) {
	typesIndex := map[string]reflect.Type{}
	for i, extraType := range extraTypes {
		typesIndex[extraType.String()] = extraTypes[i]
	}

	expr, err := parser.ParseExpr(dataType)
	if err != nil {
		return nil, err
	}

	rType, err := matchType(expr, typesIndex, dataType, shouldUnquote)
	if err != nil {
		return nil, err
	}

	return rType, nil
}

func matchType(expr ast.Expr, typesIndex map[string]reflect.Type, dataType string, shouldUnquote bool) (reflect.Type, error) {
	switch actual := expr.(type) {
	case *ast.StarExpr:
		rType, err := matchType(actual.X, typesIndex, dataType, shouldUnquote)
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

			fieldType, err := matchType(field.Type, typesIndex, dataType, shouldUnquote)
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
		return matchType(&ast.Ident{Name: dataType[actual.Pos()-1 : actual.End()-1]}, typesIndex, dataType, shouldUnquote)
	case *ast.ArrayType:
		rType, err := matchType(actual.Elt, typesIndex, dataType, shouldUnquote)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(rType), nil
	case *ast.InterfaceType:
		return InterfaceType, nil
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
			rType, ok := typesIndex[actual.Name]
			if !ok {
				return nil, fmt.Errorf("not found type %v", actual.Name)
			}

			return rType, nil
		}
	}

	return nil, fmt.Errorf("unsupported %T, %v", expr, expr)
}
