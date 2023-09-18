package xreflect

import (
	"fmt"
	"go/ast"
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

// TypeName package qualified type name
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

	if t.PackagePath != "" {
		pkg := registry.ensurePackage(t.Package, t.PackagePath)
		if pkg.dirType == nil {
			var err error
			if pkg.dirType, err = ParseTypes(t.PackagePath, WithTypeLookup(t.Registry.Lookup)); err != nil {
				return nil, err
			}
		}
		name := rawName(t.Name)
		rType, err := pkg.dirType.Type(name)
		if err != nil {
			return nil, err
		}
		if methods := pkg.dirType.Methods(name); len(methods) > 0 {
			for _, item := range methods {
				method := AsMethod(item)
				pkg.methods[t.Name] = append(pkg.methods[t.Name], method)
			}
		}

		return rType, nil
	}

	if t.Definition != "" {
		return Parse(t.Definition, WithRegistry(t.Registry))
	}
	return nil, fmt.Errorf("unable to load type: %v", t.TypeName())
}

func AsMethod(item *ast.FuncDecl) reflect.Method {
	methodName, _ := Node{item.Name}.Stringify()
	method := reflect.Method{
		Name:    methodName,
		PkgPath: "",
		Type:    nil,
		Func:    reflect.Value{},
		Index:   0,
	}
	return method
}

// NewType crates a type spec with option
func NewType(name string, opts ...Option) *Type {
	o := &options{}
	name = strings.TrimSpace(name)
	o.Type.Name = name
	o.Apply(opts...)

	isPtr := len(name) > 0 && name[0] == '*'
	if isPtr {
		name = name[1:]
	}

	if index := strings.LastIndex(name, "."); index != -1 && !strings.Contains(name, " ") {
		o.Type.Package = name[:index]
		o.Type.Name = name[index+1:]
	}

	if isPtr {
		o.Type.Name = "*" + o.Type.Name
	}

	if o.Definition == "" && (strings.Contains(o.Type.Name, "{") ||
		strings.Contains(o.Type.Name, "[") ||
		strings.Contains(o.Type.Name, "*")) {
		o.Definition = name
	}

	return &o.Type
}

func rawName(name string) string {
	if strings.HasPrefix(name, "[]") {
		name = name[2:]
	}
	if strings.HasPrefix(name, "*") {
		name = name[1:]
	}
	return name

}
