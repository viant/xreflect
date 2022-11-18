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
}

func NewDirTypes(path string) *DirTypes {
	return &DirTypes{
		path:    path,
		types:   map[string]reflect.Type{},
		subDirs: map[string]*DirTypes{},
		specs:   map[string]*ast.TypeSpec{},
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

func (t *DirTypes) lookupType(path string, identifier string, name string) (reflect.Type, bool) {
	rType, err := t.Type(name)
	return rType, err == nil
}
