package ast

import (
	"fmt"
	"go/ast"
	"go/parser"
	"strings"
)

func ParseExpr(expr string) (ast.Node, error) {
	if !strings.HasPrefix(strings.TrimSpace(expr), "func") {
		expr = `func() {` + expr + `}`
	}
	tree, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %s, %w", err, err)
	}
	return tree.(*ast.FuncLit), nil
}
