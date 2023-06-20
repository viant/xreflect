package xreflect

import (
	"fmt"
	"go/ast"
	"strings"
)

//CommentGroup extends ast.CommentGroup
type CommentGroup ast.CommentGroup

//Stringify stringifies comments
func (c CommentGroup) Stringify() string {
	if len(c.List) == 0 {
		return ""
	}
	sb := strings.Builder{}
	for _, item := range c.List {
		sb.WriteString(item.Text)
	}
	return sb.String()
}

type Node struct {
	ast.Node
}

func (n Node) Stringify() (string, error) {
	builder := strings.Builder{}
	err := stringify(n.Node, &builder)
	return builder.String(), err
}

func stringify(expr ast.Node, builder *strings.Builder) error {
	switch actual := expr.(type) {
	case *ast.BasicLit:
		builder.WriteString(actual.Value)
	case *ast.Ident:
		builder.WriteString(actual.Name)
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
	case *ast.StructType:
		builder.WriteString("struct{")
		for _, field := range actual.Fields.List {
			builder.WriteString(field.Names[0].Name)
			builder.WriteString(" ")

			stringify(field.Type, builder)
		}
		builder.WriteString("}")

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
	default:
		return fmt.Errorf("unsupported node: %T", actual)
	}
	return nil
}
