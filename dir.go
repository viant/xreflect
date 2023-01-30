package xreflect

import (
	"fmt"
	"go/ast"
	"reflect"
)

type DirTypes struct {
	types   map[string]reflect.Type
	subDirs map[string]*DirTypes
	path    string
	specs   map[string]*ast.TypeSpec
	values  map[string]interface{}
	scopes  []*ast.Scope
}

func NewDirTypes(path string) *DirTypes {
	return &DirTypes{
		path:    path,
		types:   map[string]reflect.Type{},
		subDirs: map[string]*DirTypes{},
		specs:   map[string]*ast.TypeSpec{},
		values:  map[string]interface{}{},
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

func (t *DirTypes) Value(name string) (interface{}, error) {
	if value, ok := t.values[name]; ok {
		return value, nil
	}

	for _, scope := range t.scopes {
		if anObject := scope.Lookup(name); anObject != nil {

			spec, ok := anObject.Decl.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, value := range spec.Values {
				return value, nil
			}
		}
	}
	return nil, fmt.Errorf("not found value %v", name)
}

func (t *DirTypes) addScope(scope *ast.Scope) {
	if scope == nil {
		return
	}

	t.scopes = append(t.scopes, scope)
}
