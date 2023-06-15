package xreflect

import (
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
	"strings"
)

type (
	DirTypes struct {
		options
		types            map[string]reflect.Type
		subDirs          map[string]*DirTypes
		path             string
		specs            map[string]*TypeSpec
		values           map[string]interface{}
		methods          map[string]*Methods
		scopes           map[string]*ast.Scope
		imports          map[string][]string
		typesOccurrences map[string][]string
	}

	Methods struct {
		receiver string
		methods  []*ast.FuncDecl
	}

	TypeSpec struct {
		path string
		spec *ast.TypeSpec
	}
)

func NewDirTypes(path string) *DirTypes {
	ret := &DirTypes{
		path:             path,
		types:            map[string]reflect.Type{},
		subDirs:          map[string]*DirTypes{},
		specs:            map[string]*TypeSpec{},
		values:           map[string]interface{}{},
		methods:          map[string]*Methods{},
		imports:          map[string][]string{},
		scopes:           map[string]*ast.Scope{},
		typesOccurrences: map[string][]string{},
	}
	return ret
}

func (t *DirTypes) lookup(packagePath, packageIdentifier, typeName string) (reflect.Type, error) {
	if t.options.lookup != nil {
		return t.options.lookup(packagePath, packageIdentifier, typeName)
	}
	return t.lookupType(packagePath, packageIdentifier, typeName)
}

func (t *DirTypes) registerTypeSpec(path string, spec *ast.TypeSpec) {
	t.specs[spec.Name.Name] = &TypeSpec{
		path: path,
		spec: spec,
	}
	t.typesOccurrences[spec.Name.Name] = append(t.typesOccurrences[spec.Name.Name], path)
}

func (t *DirTypes) Type(name string) (reflect.Type, error) {
	if rType, ok := t.types[name]; ok {
		return rType, nil
	}

	spec, ok := t.specs[name]
	if !ok {
		return nil, fmt.Errorf("not found type %v", name)
	}

	matched, err := t.matchType(spec.spec, false)
	if err != nil {
		return nil, err
	}

	t.types[name] = matched
	return matched, nil
}

func (t *DirTypes) lookupType(packagePath string, packageName string, name string) (reflect.Type, error) {
	if t.options.lookup != nil {
		lookup, err := t.lookup(packagePath, packageName, name)
		if err == nil {
			return lookup, nil
		}
	}
	rType, err := t.Type(name)
	return rType, err
}

func (t *DirTypes) Value(value string) (interface{}, error) {
	if value, ok := t.values[value]; ok {
		return value, nil
	}

	for _, scope := range t.scopes {
		aValue, ok := t.valueInScope(value, scope)
		if ok {
			return aValue, nil
		}
	}
	return nil, t.notFoundValueError(value)
}

func (t *DirTypes) notFoundValueError(value string) error {
	return fmt.Errorf("not found value %v", value)
}

func (t *DirTypes) valueInScope(name string, scope *ast.Scope) (interface{}, bool) {
	if anObject := scope.Lookup(name); anObject != nil {

		spec, ok := anObject.Decl.(*ast.ValueSpec)
		if !ok {
			return nil, false
		}

		for _, value := range spec.Values {
			return value, true
		}
	}
	return nil, false
}

func (t *DirTypes) addScope(path string, scope *ast.Scope) {
	if scope == nil {
		return
	}

	t.scopes[path] = scope
}

func (t *DirTypes) Methods(receiver string) []*ast.FuncDecl {
	if methods, ok := t.methods[receiver]; ok {
		return methods.methods
	}

	return nil
}

func (t *DirTypes) registerMethod(receiver string, spec *ast.FuncDecl) {
	index, ok := t.methods[receiver]
	if !ok {
		index = &Methods{
			receiver: receiver,
		}

		t.methods[receiver] = index
	}

	index.methods = append(index.methods, spec)
}

func (t *DirTypes) addImports(path string, file *ast.File) error {
	var imports []string
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return err
		}

		imports = append(imports, value)
	}

	t.imports[path] = imports
	return nil
}

func (t *DirTypes) Imports(path string) []string {
	if strings.HasPrefix(path, "*") {
		path = path[1:]
		var imports []string

		for fileName, fileImports := range t.imports {
			if strings.HasSuffix(fileName, path) {
				imports = append(imports, fileImports...)
			}
		}

		return imports
	}

	return t.imports[path]
}

func (t *DirTypes) ValueInFile(file, value string) (interface{}, error) {
	scope, ok := t.scopes[file]
	if !ok {
		return nil, fmt.Errorf("not found file %v", file)
	}

	aValue, ok := t.valueInScope(value, scope)
	if ok {
		return aValue, nil
	}

	return nil, t.notFoundValueError(value)
}

func (t *DirTypes) TypesOccurrences(typeName string) []string {
	return t.typesOccurrences[typeName]
}
