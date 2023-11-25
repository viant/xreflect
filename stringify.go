package xreflect

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
)

// CommentGroup extends ast.CommentGroup
type CommentGroup ast.CommentGroup

// Stringify stringifies comments
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

func Stringify(rType reflect.Type, tag reflect.StructTag) string {
	builder := &strings.Builder{}
	stringifyWithBuilder(rType, tag, builder)
	return builder.String()
}

func stringifyWithBuilder(rType reflect.Type, tag reflect.StructTag, builder *strings.Builder) {
	aliasedTypeName := tag.Get(TagTypeName)
	if aliasedTypeName == "" {
		builder.WriteString(rType.String())
		return
	}
	for {
		switch rType.Kind() {
		case reflect.Ptr:
			builder.WriteByte('*')
			rType = rType.Elem()
		case reflect.Slice, reflect.Array:
			builder.WriteString("[]")
			rType = rType.Elem()
		default:
			builder.WriteString(aliasedTypeName)
			return
		}
	}
}

func baseType(rType reflect.Type) reflect.Type {
	switch rType.Kind() {
	case reflect.Ptr:
		return baseType(rType.Elem())
	case reflect.Slice, reflect.Array:
		return baseType(rType.Elem())
	default:
		return rType
	}
}
func (t *Type) String() string {
	builder := strings.Builder{}
	tag := reflect.StructTag(TagTypeName + `:"` + t.Name + `"`)
	builder.WriteString("type ")
	t.stringifyWithBuilder(t.Type, tag, &builder)
	builder.WriteString(" ")
	return t.body(&builder)
}

func (t *Type) Body() string {
	builder := strings.Builder{}
	t.body(&builder)
	return builder.String()
}

func (t *Type) body(builder *strings.Builder) string {
	t.stringify(t.Type, "", builder)
	return builder.String()
}

func (t *Type) stringify(rType reflect.Type, tag reflect.StructTag, builder *strings.Builder) {
	bType := baseType(rType)
	switch bType.Kind() {
	case reflect.Interface:
		builder.WriteString("interface{}")
	case reflect.Struct:
		if bType.Name() != "" {
			return
		}
		builder.WriteString("struct{")
		for i := 0; i < bType.NumField(); i++ {
			aField := bType.Field(i)
			fieldTag := string(aField.Tag)
			isIface := hasInterface(aField.Type)
			if !isIface { //preserve type name for interface type
				fieldTag, _ = removeTag(string(aField.Tag), TagTypeName)
			}
			isNamedType := aField.Type.Name() != "" || aField.Tag.Get(TagTypeName) != ""
			if !aField.Anonymous {
				builder.WriteString(aField.Name)
			}
			builder.WriteString(" ")
			t.stringifyWithBuilder(aField.Type, aField.Tag, builder)
			if !isNamedType {
				t.stringify(aField.Type, aField.Tag, builder)
			}
			if aField.Tag != "" {
				builder.WriteString(" `")
				builder.WriteString(fieldTag)
				builder.WriteString("`")
			}
			builder.WriteString("; ")
		}
		builder.WriteString("}")
	}
}

func hasInterface(aType reflect.Type) bool {
	switch aType.Kind() {
	case reflect.Ptr:
		return hasInterface(aType.Elem())
	case reflect.Slice:
		return hasInterface(aType.Elem())
	case reflect.Interface:
		return true
	}
	return false
}

func removeTag(tag string, tagName string) (string, string) {
	tag = strings.TrimSpace(tag)
	tag = trim(tag, '`')
	tag = " " + tag
	fragment := ""
	tagName = ` ` + tagName
	tagName += ":"
	if index := strings.Index(tag, tagName); index != -1 {
		matched := tag[index:]
		offset := len(tagName) + 1
		if index := strings.Index(matched[offset:], `"`); index != -1 {
			matched = matched[:offset+index+1]
			fragment = strings.Trim(matched[offset:], `"`)
			tag = strings.Replace(tag, matched, "", 1)
		}
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", ""
	}
	return tag, fragment
}

func (t *Type) stringifyWithBuilder(rType reflect.Type, tag reflect.StructTag, builder *strings.Builder) {
	typeName := tag.Get(TagTypeName)
	if rType.Name() != "" {
		builder.WriteString(t.namedType(rType))
		return
	}
	for {
		switch rType.Kind() {
		case reflect.Ptr:
			builder.WriteByte('*')
			rType = rType.Elem()
			if rType.Name() != "" {
				builder.WriteString(t.namedType(rType))
				return
			}
		case reflect.Slice, reflect.Array:
			builder.WriteString("[]")
			rType = rType.Elem()
		default:
			if typeName == "" {
				typeName = t.namedType(rType)
			}
			builder.WriteString(typeName)
			return
		}
	}
}

func (t *Type) namedType(rType reflect.Type) string {
	pkg := relativePackage(rType)
	if pkg != "" && pkg != t.Package {
		return pkg + "." + rType.Name()
	}
	return rType.Name()
}

func trim(tag string, c byte) string {
	if tag == "" {
		return ""
	}
	if tag[0] == c && tag[len(tag)-1] == c {
		tag = tag[1 : len(tag)-1]
	}
	return tag
}

func (t *Type) relativePackage(rType reflect.Type) string {
	pkg := relativePackage(rType)
	if pkg == "" {
		return ""
	}
	if pkg == t.Package {
		return ""
	}
	return pkg
}

func relativePackage(rType reflect.Type) string {
	pkg := rType.PkgPath()
	if pkg == "" {
		return ""
	}
	index := strings.LastIndex(pkg, "/")
	if index != -1 {
		pkg = pkg[index+1:]
	}
	return pkg
}
