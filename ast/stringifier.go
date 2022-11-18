package ast

import (
	"fmt"
	"go/ast"
	"go/format"
	"strings"
)

//StringifyExpr returns sting representation of expression
func StringifyExpr(expr ast.Node) (string, error) {
	builder := strings.Builder{}
	err := stringify(expr, &builder)
	if err != nil {
		return "", err
	}
	output := builder.String()
	formatted, err := format.Source([]byte(output))
	if err != nil {
		return output, err
	}
	return string(formatted), err
}

func stringify(expr ast.Node, builder *strings.Builder) error {
	switch actual := expr.(type) {
	case *ast.BasicLit:
		builder.WriteString(actual.Value)
	case *ast.DeclStmt:
		if err := stringify(actual.Decl, builder); err != nil {
			return err
		}
	case *ast.Ident:
		builder.WriteString(actual.Name)
	case *ast.GenDecl:

		builder.WriteString(actual.Tok.String())
		builder.WriteString(" ")
		for _, s := range actual.Specs {
			if err := stringify(s, builder); err != nil {
				return err
			}
			builder.WriteString(" ")
		}
	case *ast.TypeSpec:
		if err := stringify(actual.Name, builder); err != nil {
			return err
		}
		builder.WriteString(" ")

		return stringify(actual.Type, builder)
	case *ast.IndexExpr:
		if err := stringify(actual.X, builder); err != nil {
			return err
		}
		builder.WriteString("[")
		if err := stringify(actual.Index, builder); err != nil {
			return err
		}
		builder.WriteString("]")
	case *ast.SelectorExpr:
		if err := stringify(actual.X, builder); err != nil {
			return err
		}
		builder.WriteString(".")
		return stringify(actual.Sel, builder)
	case *ast.ParenExpr:
		builder.WriteString("(")
		if err := stringify(actual.X, builder); err != nil {
			return err
		}
		builder.WriteString(")")
	case *ast.CallExpr:
		if err := stringify(actual.Fun, builder); err != nil {
			return err
		}
		builder.WriteString("(")
		for i := 0; i < len(actual.Args); i++ {
			if i > 0 {
				builder.WriteString(",")
			}
			if err := stringify(actual.Args[i], builder); err != nil {
				return err
			}
		}
		builder.WriteString(")")
	case *ast.BinaryExpr:
		if err := stringify(actual.X, builder); err != nil {
			return err
		}
		builder.WriteString(actual.Op.String())
		return stringify(actual.Y, builder)
	case *ast.UnaryExpr:
		builder.WriteString(actual.Op.String())
		return stringify(actual.X, builder)
	case *ast.ArrayType:
		builder.WriteString("[]")
		return stringify(actual.Elt, builder)
	case *ast.StarExpr:
		builder.WriteString("*")
		return stringify(actual.X, builder)
	case *ast.CompositeLit:
		if err := stringify(actual.Type, builder); err != nil {
			return err
		}
		builder.WriteString("{}")
	case *ast.StructType:
		if err := stringifyStructType(builder, actual); err != nil {
			return err
		}

	case *ast.AssignStmt:
		if err := stringifyList(actual.Lhs, builder); err != nil {
			return err
		}
		builder.WriteString(" := ")
		if err := stringifyList(actual.Rhs, builder); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported node: %T", actual)
	}
	return nil
}

func stringifyStructType(builder *strings.Builder, structType *ast.StructType) error {

	builder.WriteString("struct{\n")
	for i, f := range structType.Fields.List {
		if i > 0 {
			builder.WriteString("\n")
		}
		for j, id := range f.Names {
			if j > 0 {
				builder.WriteString(",")
			}
			if err := stringify(id, builder); err != nil {
				return err
			}
		}
		builder.WriteString(" ")
		if err := stringify(f.Type, builder); err != nil {
			return err
		}
		if f.Tag != nil {
			tag := strings.Trim(f.Tag.Value, "\"")
			builder.WriteString(" `" + tag + "`")
		}

	}
	builder.WriteString("}")
	return nil
}

func stringifyList(list []ast.Expr, builder *strings.Builder) error {
	for i, item := range list {
		if i > 1 {
			builder.WriteString(",")
		}
		if err := stringify(item, builder); err != nil {
			return err
		}
	}
	return nil
}
