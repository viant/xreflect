package xreflect

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
)

const customPackageName = "PackageName"

type Type struct {
	PackagePath string
	ModulePath  string
	Package     string
	Name        string
	KeyName     string
	KeyType     reflect.Type
	Definition  string
	Type        reflect.Type
	Methods     []reflect.Method
	Registry    *Types
	IsPtr       bool
}

// TypeName package qualified type name
func (t *Type) SimpleTypeName() string {
	return t.Name
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

	var err error
	var rType reflect.Type
	name := rawName(t.Name)

	if t.PackagePath != "" {
		pkg := registry.ensurePackage(t.Package, t.PackagePath)

		if pkg.dirType == nil {
			dirType, err := ParseTypes(t.PackagePath, WithTypeLookup(t.Registry.Lookup))
			if err != nil {
				return nil, err
			}
			rType, err = dirType.Type(name)
			if err != nil {
				return nil, err
			}
			t.ModulePath = dirType.ModulePath
			packageName := dirType.PackagePath(t.PackagePath) //ensure location package matches actual package
			if value, err := dirType.Value(customPackageName); err == nil {
				if literal, ok := value.(*ast.BasicLit); ok {
					if customPackage := strings.Trim(literal.Value, `"`); customPackage != packageName {
						packageName = customPackage
					}
				}
			}
			if packageName != "" && packageName != pkg.Name { //otherwise correct it
				pkg.packagePaths[t.PackagePath] = packageName
				pkg.Path = ""
				pkg = registry.ensurePackage(packageName, t.PackagePath)
			}
			pkg.dirType = dirType
		} else {
			rType, err = pkg.dirType.Type(name)
			if err != nil {
				return nil, err
			}
		}
		t.Package = pkg.Name
		if methods := pkg.dirType.Methods(name); len(methods) > 0 {
			for _, item := range methods {
				method := AsMethod(item)
				pkg.methods[t.Name] = append(pkg.methods[t.Name], method)
				t.Methods = append(t.Methods, method)
			}
		}
		return rType, nil
	}
	if t.Definition != "" {
		return Parse(t.Definition, WithRegistry(t.Registry), WithPackage(t.Package))
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
	isMap, key, name := isMap(name)
	if isMap {
		o.KeyName = key
	}
	sliceDef, name := isSlice(name)
	isPtr, name := isPointer(name)
	o.Apply(opts...)
	if index := strings.LastIndex(name, "."); index != -1 && !strings.Contains(name, " ") {
		o.Type.Package = name[:index]
		name = name[index+1:]
	}

	if sliceDef {
		o.Type.Name = "[]"
	}
	if isPtr {
		o.Type.Name += "*" + name
	} else {
		o.Type.Name += name
	}

	if o.Definition == "" && (strings.Contains(o.Type.Name, "{") ||
		strings.Contains(o.Type.Name, "[") ||
		strings.Contains(o.Type.Name, "*")) {
		o.Definition = name
	}
	return &o.Type
}

func isPointer(name string) (bool, string) {
	isPtr := len(name) > 0 && name[0] == '*'
	if isPtr {
		name = name[1:]
	}
	return isPtr, name
}

func isMap(name string) (bool, string, string) {
	isMap := len(name) > 4 && name[0:4] == "map["
	key := ""
	if isMap {
		key = name[4:]
		if index := strings.Index(key, "]"); index != -1 {
			name = key[index+1:]
			key = key[:index]
		}
	}
	return isMap, key, name
}

func isSlice(name string) (bool, string) {
	isPtr := len(name) > 1 && name[0:2] == "[]"
	if isPtr {
		name = name[2:]
	}
	return isPtr, name
}

func componentType(name string) string {
	if strings.HasPrefix(name, "[]") {
		name = name[2:]
	}
	if strings.HasPrefix(name, "*") {
		name = name[1:]
	}
	return name
}

func rawName(name string) string {
	name = componentType(name)
	if index := strings.LastIndex(name, "."); index != -1 {
		name = name[index+1:]
	}
	return name

}
