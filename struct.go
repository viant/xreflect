package xreflect

import (
	"go/format"
	"reflect"
	"strings"
)

func GenerateStruct(name string, structType reflect.Type) string {
	typeBuilder := newTypeBuilder(name)
	importsBuilder := &strings.Builder{}

	dependencyTypes := buildGoType(typeBuilder, importsBuilder, structType)

	generated := build(importsBuilder, typeBuilder, dependencyTypes)
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

func build(importsBuilder *strings.Builder, structBuilder *strings.Builder, types []*strings.Builder) string {
	result := strings.Builder{}
	result.WriteString("package generated \n\n")

	if importsBuilder.Len() > 0 {
		result.WriteString("import (\n")
		result.WriteString(importsBuilder.String())
		result.WriteString(")\n\n")
	}

	result.WriteString(structBuilder.String())

	for _, builder := range types {
		result.WriteString("\n\n")
		result.WriteString(builder.String())
	}

	return result.String()
}

func buildGoType(mainBuilder *strings.Builder, importsBuilder *strings.Builder, structType reflect.Type) []*strings.Builder {
	structType = appendElem(mainBuilder, structType)
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
					structBuilders = append(structBuilders, buildGoType(nestedStruct, importsBuilder, actualType)...)
				} else {
					mainBuilder.WriteString(actualType.String())
					importsBuilder.WriteString(`  "`)
					importsBuilder.WriteString(actualType.PkgPath())
					importsBuilder.WriteByte('"')
					importsBuilder.WriteByte('\n')
				}
			} else {
				structBuilders = append(structBuilders, buildGoType(mainBuilder, importsBuilder, actualType)...)
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
