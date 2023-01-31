package xreflect

import (
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
)

type (
	DirTypes struct {
		types   map[string]reflect.Type
		subDirs map[string]*DirTypes
		path    string
		specs   map[string]*ast.TypeSpec
		values  map[string]interface{}
		methods map[string]*Methods
		scopes  map[string]*ast.Scope
		imports map[string][]string
	}

	Methods struct {
		receiver string
		methods  []*ast.FuncDecl
	}
)

func NewDirTypes(path string) *DirTypes {
	return &DirTypes{
		path:    path,
		types:   map[string]reflect.Type{},
		subDirs: map[string]*DirTypes{},
		specs:   map[string]*ast.TypeSpec{},
		values:  map[string]interface{}{},
		methods: map[string]*Methods{},
		imports: map[string][]string{},
		scopes:  map[string]*ast.Scope{},
	}
}

func (t *DirTypes) indexTypeSpec(spec *ast.TypeSpec) {
	t.specs[spec.Name.Name] = spec
}

func (t *DirTypes) Type(name string) (reflect.Type, error) {
	if rType, ok := t.types[name]; ok {
		return rType, nil
	}

	spec, ok := t.specs[name]
	if !ok {
		return nil, fmt.Errorf("not found type %v", name)
	}

	matched, err := matchType(spec, false, t.lookupType)
	if err != nil {
		return nil, err
	}

	t.types[name] = matched
	return matched, nil
}

func (t *DirTypes) lookupType(_ string, _ string, name string) (reflect.Type, bool) {
	rType, err := t.Type(name)
	return rType, err == nil
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
