package xreflect

import (
	"fmt"
	"reflect"
	"strings"
)

type Type struct {
	PackagePath string
	Package     string
	Name        string
	Definition  string
	Type        reflect.Type
	Registry    *Types
}

//TypeName package qualified type name
func (t *Type) TypeName() string {
	if t.Package == "" {
		return t.Name
	}
	return t.Package + "." + t.Name
}

func (t *Type) IsLoadable() bool {
	return t.Definition != "" || t.PackagePath != ""
}

func (t *Type) LoadType(registry *Types) (reflect.Type, error) {
	if t.Registry == nil {
		t.Registry = registry
	}
	registry = t.Registry
	if t.Definition != "" {
		return Parse(t.Definition, WithRegistry(registry))
	}
	if t.PackagePath != "" {
		pkg := registry.ensurePackage(t.Package, t.PackagePath)
		if pkg.dirType == nil {
			pkg.dirType = NewDirTypes(t.PackagePath)
		}
		return pkg.dirType.Type(t.Name)
	}
	return nil, fmt.Errorf("unable to load type: %v", t.TypeName())
}

//NewType crates a type spec with option
func NewType(name string, opts ...Option) *Type {
	o := &options{}
	name = strings.TrimSpace(name)
	o.Type.Name = name
	o.Apply(opts...)

	if index := strings.LastIndex(name, "."); index != -1 && !strings.Contains(name, " ") {
		o.Type.Package = name[:index]
		o.Type.Name = name[index+1:]
	}
	if o.Definition == "" && (strings.Contains(o.Type.Name, "{") ||
		strings.Contains(o.Type.Name, "[") ||
		strings.Contains(o.Type.Name, "*")) {
		o.Definition = name
	}

	return &o.Type
}
