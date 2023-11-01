package xreflect

import (
	"go/format"
	"reflect"
	"strconv"
	"strings"
)

type (
	Imports          []string
	AppendBeforeType string
	PackageName      string
)

func GenerateStruct(name string, structType reflect.Type, opts ...Option) string {
	genOptions := &options{}
	genOptions.Apply(opts...)
	genOptions.initGen()
	typeBuilder := newTypeBuilder(name)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	importsBuilder := &strings.Builder{}

	for _, imported := range genOptions.imports {
		importsBuilder.WriteByte('"')
		importsBuilder.WriteString(imported)
		importsBuilder.WriteString("\"\n")
	}

	dependencyTypes := buildGoType(typeBuilder, importsBuilder, structType, map[string]bool{}, true, genOptions)

	additionalTypeBuilder := strings.Builder{}
	for _, aType := range genOptions.withTypes {
		additionalTypeBuilder.WriteString("\n\n")
		aTypeBuilder := newTypeBuilder(aType.Name)
		dep := buildGoType(aTypeBuilder, importsBuilder, aType.Type, map[string]bool{}, true, genOptions)
		additionalTypeBuilder.WriteString(aTypeBuilder.String())
		for _, builder := range dep {
			additionalTypeBuilder.WriteString("\n\n")
			additionalTypeBuilder.WriteString(builder.String())
		}
	}

	generated := build(importsBuilder, typeBuilder, dependencyTypes, genOptions.snippetBefore, genOptions.Package)
	generated += additionalTypeBuilder.String()
	if genOptions.snippetAfter != "" {
		generated += genOptions.snippetAfter
	}
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

func buildGoType(mainBuilder *strings.Builder, importsBuilder *strings.Builder, structType reflect.Type, imports map[string]bool, isMain bool, opts *options) []*strings.Builder {
	structType = appendElem(mainBuilder, structType)
	appendImportIfNeeded(importsBuilder, structType.PkgPath(), imports, isMain, opts)

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
					pkgType := opts.generateOption.getPackageType(typeName)
					if pkgType != nil {
						appendImportIfNeeded(importsBuilder, pkgType.Package, imports, false, opts)
						pkgTypeName := pkgType.Package + "." + typeName
						mainBuilder.WriteString(pkgTypeName)

					} else {
						mainBuilder.WriteString(typeName)
						nestedStruct := &strings.Builder{}
						structBuilders = append(structBuilders, nestedStruct)
						nestedStruct.WriteString("type ")
						nestedStruct.WriteString(typeName)
						nestedStruct.WriteByte(' ')
						structBuilders = append(structBuilders, buildGoType(nestedStruct, importsBuilder, actualType, imports, false, opts)...)
					}
				} else {
					mainBuilder.WriteString(actualType.String())
					appendImportIfNeeded(importsBuilder, actualType.PkgPath(), imports, false, opts)
				}
			} else {
				structBuilders = append(structBuilders, buildGoType(mainBuilder, importsBuilder, actualType, imports, false, opts)...)
			}
			tagValue := aField.Tag
			if tagValue != "" {
				quoteChar := "`"
				if strings.Contains(string(aField.Tag), "`") {
					quoteChar = `"`
					tagValue = reflect.StructTag(strconv.Quote(string(aField.Tag)))
				}

				mainBuilder.WriteByte(' ')
				mainBuilder.WriteString(quoteChar)
				mainBuilder.WriteString(string(tagValue))
				mainBuilder.WriteString(quoteChar)
			}
		}
		mainBuilder.WriteString("\n}")

	default:
		mainBuilder.WriteString(structType.String())
	}

	return structBuilders
}

func appendImportIfNeeded(importsBuilder *strings.Builder, pkgPath string, imports map[string]bool, isMain bool, opts *options) {
	if isMain {
		return
	}

	if pkgPath == "" || imports[pkgPath] {
		return
	}

	imports[pkgPath] = true
	importsBuilder.WriteString(`  "`)
	if len(opts.importModule) > 0 {
		if modulePath, ok := opts.importModule[pkgPath]; ok {
			importsBuilder.WriteString(modulePath)
			importsBuilder.WriteByte('/')
		}
	}
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
