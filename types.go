package xreflect

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"sync"
)

type LookupType func(name string, option ...Option) (reflect.Type, error)

type Types struct {
	mux      sync.RWMutex
	parent   *Types
	packages map[string]*Package
	info     map[reflect.Type]*Type
}

func (t *Types) PackageNames() []string {
	var result []string
	t.mux.RLock()
	for k := range t.packages {
		result = append(result, k)
	}
	t.mux.RUnlock()
	return result
}

func (t *Types) Has(name string) bool {
	aType := NewType(name)
	pkg := t.Package(aType.Package)
	if pkg == nil {
		return false
	}
	ret, _ := pkg.Lookup(aType.Name)
	return ret != nil
}

func (t *Types) SetParent(parent *Types) {
	t.parent = parent
}

func (t *Types) Info(rt reflect.Type) *Type {
	t.mux.RLock()
	ret := t.info[rt]
	t.mux.RUnlock()
	return ret
}

func (t *Types) Package(name string) *Package {
	t.mux.RLock()
	pkg := t.packages[name]
	t.mux.RUnlock()
	return pkg
}

func (t *Types) MergeFrom(from *Types) error {
	if from == nil {
		return nil
	}
	packages := from.PackageNames()
	for _, pkgName := range packages {
		pkg := from.Package(pkgName)
		destPkg := t.ensurePackage(pkg.Name, pkg.Path)
		typeNames := pkg.TypeNames()
		for _, name := range typeNames {
			aType, _ := pkg.Lookup(name)
			if err := destPkg.register(name, aType); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Types) Methods(name string, opts ...Option) ([]reflect.Method, error) {
	aType := NewType(name, opts...)
	ret, err := t.lookupMethods(aType)
	if ret != nil {
		return ret, nil
	}
	if t.parent != nil {
		return t.parent.Methods(name, opts...)
	}
	return nil, err
}

func (t *Types) Symbol(symbol string, opts ...Option) (interface{}, error) {
	aType := NewType("", opts...)
	pkg := t.ensurePackage(aType.Package, aType.PackagePath)
	var err error
	if pkg.dirType == nil {
		if pkg.dirType, err = ParseTypes(pkg.Path, WithTypeLookup(t.Lookup)); err != nil {
			return nil, err
		}
	}
	val, err := pkg.dirType.Value(symbol)
	if err != nil {
		return nil, err
	}
	switch actual := val.(type) {
	case *ast.BasicLit:
		value := actual.Value
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = value[1 : len(value)-1]
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", val)
	}
}

func (t *Types) Lookup(name string, opts ...Option) (reflect.Type, error) {
	aType := NewType(name, opts...)
	return t.LookupType(aType)
}

func (t *Types) LookupType(aType *Type) (reflect.Type, error) {
	if aType.Name == "Time" {
		fmt.Printf("")
	}
	ret, err := t.lookupType(aType)
	if err != nil && t.parent != nil {
		if ret, _ = t.parent.LookupType(aType); ret != nil {
			return ret, nil
		}
	}
	return ret, err
}

func (t *Types) lookupMethods(aType *Type) ([]reflect.Method, error) {
	t.mux.RLock()
	pkg := t.packages[aType.Package]
	t.mux.RUnlock()
	if pkg == nil {
		if !aType.IsLoadable() {
			return nil, fmt.Errorf("unable locate: %s unknown package: '%s'", aType.Name, aType.Package)
		}
		pkg = t.ensurePackage(aType.Package, aType.PackagePath)
	}

	return pkg.Methods(aType.Name)
}

func (t *Types) lookupType(aType *Type) (reflect.Type, error) {
	t.mux.RLock()
	pkg := t.packages[aType.Package]
	t.mux.RUnlock()
	if pkg == nil {
		if !aType.IsLoadable() {
			return nil, fmt.Errorf("unable locate: %s unknown package: '%s'", aType.Name, aType.Package)
		}
		pkg = t.ensurePackage(aType.Package, aType.PackagePath)
	}

	rType, err := pkg.Lookup(aType.Name)
	if err != nil && aType.IsLoadable() {
		_ = t.registerType(aType)
		rType, err = pkg.Lookup(aType.Name)
	}
	return rType, err
}

func (t *Types) Register(name string, opts ...Option) error {
	opts = append([]Option{WithRegistry(t)}, opts...)
	aType := NewType(name, opts...)
	return t.registerType(aType)
}

func (t *Types) RegisterReflectTypes(types []reflect.Type, opts ...Option) error {
	for _, rType := range types {
		aType := NewType(rType.Name(), opts...)
		if err := t.registerType(aType); err != nil {
			return err
		}
	}
	return nil
}

func (t *Types) registerType(aType *Type) error {
	var err error
	t.ensurePackage(aType.Package, aType.PackagePath)
	if aType.Type == nil {
		if !aType.IsLoadable() {
			return fmt.Errorf("failed to register %v reflect.Type was nil", aType.TypeName())
		}
		if aType.Type, err = aType.LoadType(t); err != nil {
			return err
		}
	}
	t.mux.RLock()
	prev, ok := t.info[aType.Type]
	t.mux.RUnlock()
	//if previous type is a named type, it should not be overridden by inlined type i.e struct{X ...}
	if ok && prev.Type.Name() != "" && aType.Type.Name() == "" {
		return nil
	}
	t.mux.Lock()
	t.info[aType.Type] = aType
	t.mux.Unlock()

	return t.packages[aType.Package].register(aType.Name, aType.Type)
}

func (t *Types) ensurePackage(pkg string, path string) *Package {
	t.mux.RLock()
	ret, ok := t.packages[pkg]
	t.mux.RUnlock()
	if ok {
		return ret
	}
	t.mux.Lock()
	if len(t.packages) == 0 {
		t.packages = map[string]*Package{}
	}
	ret = &Package{Name: pkg, Path: path, Types: map[string]reflect.Type{}, methods: map[string][]reflect.Method{}}
	t.packages[pkg] = ret
	t.mux.Unlock()
	return ret
}

type Package struct {
	mux     sync.RWMutex
	dirType *DirTypes
	Final   bool ///final package type can not be overriden //TODO add checks with error handling
	Name    string
	Path    string
	Types   map[string]reflect.Type
	methods map[string][]reflect.Method
}

func (p *Package) TypeNames() []string {
	var result []string
	p.mux.RLock()
	for k := range p.Types {
		result = append(result, k)
	}
	p.mux.RUnlock()
	return result
}

func (p *Package) Methods(name string) ([]reflect.Method, error) {
	rType, err := p.Lookup(name)
	if err != nil {
		return nil, err
	}
	if rType == nil {
		return nil, fmt.Errorf("failed to lookup type: %v", name)
	}
	return p.methods[name], nil
}

func (p *Package) Lookup(name string) (reflect.Type, error) {
	p.mux.RLock()
	ret, ok := p.Types[name]
	p.mux.RUnlock()

	if !ok {
		if name == "Time" {
			fmt.Printf("")
		}
		if strings.HasPrefix(name, "*") {
			if ret, ok = p.Types[name[1:]]; ok {
				return reflect.PtrTo(ret), nil
			}
		}
		return nil, fmt.Errorf("unable locate : %s in package: %s", name, p.Name)
	}
	return ret, nil
}

// register registers a type in the package,
func (p *Package) register(name string, t reflect.Type) error {
	p.mux.Lock()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	p.Types[name] = t
	p.mux.Unlock()
	return nil
}

func NewTypes(opts ...Option) *Types {
	registry := &Types{packages: map[string]*Package{}, info: map[reflect.Type]*Type{}}
	o := options{}
	o.Apply(opts...)
	if o.Registry != nil {
		registry.parent = o.Registry
	}
	for _, t := range o.withReflectTypes {
		name := t.Name()
		if o.withReflectPackage != "" {
			name = o.withReflectPackage + "." + name
		}
		_ = registry.Register(name, WithReflectType(t))
	}
	for i := range o.withTypes {
		_ = registry.registerType(o.withTypes[i])
	}
	return registry
}
