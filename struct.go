package xreflect

import (
	"go/format"
	"reflect"
	"strings"
)

type (
	Imports          []string
	AppendBeforeType string
	PackageName      string
)

func GenerateStruct(name string, structType reflect.Type, options ...interface{}) string {
	var imports []string
	var appendBeforeType string
	packageName := "generated"
	for _, option := range options {
		switch actual := option.(type) {
		case Imports:
			imports = actual
		case AppendBeforeType:
			appendBeforeType = string(actual)
		case PackageName:
			packageName = string(actual)
		}
	}

	typeBuilder := newTypeBuilder(name)
	importsBuilder := &strings.Builder{}

	for _, imported := range imports {
		importsBuilder.WriteByte('"')
		importsBuilder.WriteString(imported)
		importsBuilder.WriteString("\"\n")
	}

	dependencyTypes := buildGoType(typeBuilder, importsBuilder, structType, map[string]bool{}, true)

	generated := build(importsBuilder, typeBuilder, dependencyTypes, appendBeforeType, packageName)
	source, err := format.Source([]byte(generated))
	if err == nil {
		return string(source)
	}

	return generated
}

func newTypeBuilder(name string) *strings.Builder {
	structBuilder := &strings.Builder{}
	structBuilder.WriteString("type ")
	structBuilder.WriteString(name)
	structBuilder.WriteString(" ")
	return structBuilder
}

func build(importsBuilder *strings.Builder, structBuilder *strings.Builder, types []*strings.Builder, beforeType string, packageName string) string {
	result := strings.Builder{}
	result.WriteString("package ")
	result.WriteString(packageName)
	result.WriteString("\n\n")

	if importsBuilder.Len() > 0 {
		result.WriteString("import (\n")
		result.WriteString(importsBuilder.String())
		result.WriteString(")\n\n")
	}

	if beforeType != "" {
		result.WriteString(beforeType)
		result.WriteString("\n\n")
	}

	result.WriteString(structBuilder.String())

	for _, builder := range types {
		result.WriteString("\n\n")
		result.WriteString(builder.String())
	}

	return result.String()
}

func buildGoType(mainBuilder *strings.Builder, importsBuilder *strings.Builder, structType reflect.Type, imports map[string]bool, isMain bool) []*strings.Builder {
	structType = appendElem(mainBuilder, structType)
	appendImportIfNeeded(importsBuilder, structType, imports, isMain)

	var structBuilders []*strings.Builder
	switch structType.Kind() {
	case reflect.Struct:
		numField := structType.NumField()
		mainBuilder.WriteString(" struct ")

		mainBuilder.WriteString("{")
		for i := 0; i < numField; i++ {
			mainBuilder.WriteString("\n    ")
			aField := structType.Field(i)
			mainBuilder.WriteString(aField.Name)
			mainBuilder.WriteByte(' ')
			actualType := appendElem(mainBuilder, aField.Type)
			mainBuilder.WriteByte(' ')

			if actualType.Kind() == reflect.Struct {

				if actualType.Name() == "" {
					typeName := firstNotEmptyString(aField.Tag.Get(TagTypeName), aField.Name)
					mainBuilder.WriteString(typeName)
					nestedStruct := &strings.Builder{}
					structBuilders = append(structBuilders, nestedStruct)

					nestedStruct.WriteString("type ")
					nestedStruct.WriteString(typeName)
					nestedStruct.WriteByte(' ')
					structBuilders = append(structBuilders, buildGoType(nestedStruct, importsBuilder, actualType, imports, false)...)
				} else {
					mainBuilder.WriteString(actualType.String())
					appendImportIfNeeded(importsBuilder, actualType, imports, false)
				}
			} else {
				structBuilders = append(structBuilders, buildGoType(mainBuilder, importsBuilder, actualType, imports, false)...)
			}

			if aField.Tag != "" {
				mainBuilder.WriteByte(' ')
				mainBuilder.WriteByte('`')
				mainBuilder.WriteString(string(aField.Tag))
				mainBuilder.WriteByte('`')
			}
		}
		mainBuilder.WriteString("\n}")

	default:
		mainBuilder.WriteString(structType.String())
	}

	return structBuilders
}

func appendImportIfNeeded(importsBuilder *strings.Builder, actualType reflect.Type, imports map[string]bool, isMain bool) {
	if isMain {
		return
	}

	pkgPath := actualType.PkgPath()
	if pkgPath == "" || imports[pkgPath] {
		return
	}

	imports[pkgPath] = true
	importsBuilder.WriteString(`  "`)
	importsBuilder.WriteString(pkgPath)
	importsBuilder.WriteByte('"')
	importsBuilder.WriteByte('\n')
}

func appendElem(sb *strings.Builder, rType reflect.Type) reflect.Type {
	for rType.Kind() == reflect.Ptr || rType.Kind() == reflect.Slice {
		switch rType.Kind() {
		case reflect.Ptr:
			sb.WriteByte('*')
		case reflect.Slice:
			sb.WriteString("[]")
		}

		rType = rType.Elem()
	}

	return rType
}

func firstNotEmptyString(value ...string) string {
	for _, s := range value {
		if s != "" {
			return s
		}
	}

	return ""
}
