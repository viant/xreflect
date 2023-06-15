package xreflect

import (
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
