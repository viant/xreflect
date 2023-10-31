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
		imports          map[string][]*goImport
		typesOccurrences map[string][]string
	}

	Methods struct {
		receiver string
		methods  []*ast.FuncDecl
	}

	TypeSpec struct {
		path string
		pkg  string
		spec *ast.TypeSpec
		*DirTypes
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
		imports:          map[string][]*goImport{},
		scopes:           map[string]*ast.Scope{},
		typesOccurrences: map[string][]string{},
	}
	return ret
}

func (t *TypeSpec) lookup(packagePath, packageIdentifier, typeName string) (reflect.Type, error) {
	if t.options.lookup != nil {
		rType, err := t.options.lookup(typeName, WithPackagePath(packagePath), WithPackage(packageIdentifier))
		if err == nil {
			return rType, nil
		}
	}
	rType, err := t.Type(typeName)
	if rType != nil {
		return rType, nil
	}

	if imports, ok := t.imports[t.path]; ok {
		if imp := t.matchDir(imports, packageIdentifier); imp != nil {
			location, folder := sourceLocation(t, imp)
			if location != "" {
				subDir, ok := t.subDirs[folder]
				if !ok {
					subDir, err = ParseTypes(location, withOptions(&t.options))
					if err != nil {
						return nil, err
					}
					t.subDirs[folder] = subDir
				}
				dirSpec := &TypeSpec{path: t.path, DirTypes: subDir}
				return dirSpec.lookup(packagePath, packageIdentifier, typeName)
			}
		}
	}
	return nil, err

}

func (t *DirTypes) registerTypeSpec(path string, pkg string, spec *ast.TypeSpec) {
	t.specs[spec.Name.Name] = &TypeSpec{
		path: path,
		pkg:  pkg,
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
	spec.DirTypes = t
	matched, err := spec.matchType(spec.pkg, spec.spec, spec.spec.Type)
	if err != nil {

		return nil, err
	}

	t.types[name] = matched
	return matched, nil
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

// TypesNames returns types names
func (t *DirTypes) TypesNames() []string {
	var result []string
	if len(t.specs) == 0 {
		return result
	}
	for key := range t.specs {
		result = append(result, key)
	}
	return result
}

func (t *DirTypes) TypeNamesInPath(aPath string) []string {
	var result []string
	val, ok := t.scopes[aPath]
	if !ok {
		return result
	}
	for k := range val.Objects {
		result = append(result, k)
	}
	return result
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

type goImport struct {
	Name string
	Path string
}

func (t *DirTypes) addImports(path string, file *ast.File) error {
	var imports []*goImport
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return err
		}
		imp := &goImport{Path: value}
		if spec.Name != nil {
			imp.Name = spec.Name.Name
		}
		imports = append(imports, imp)
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
				for _, imp := range fileImports {
					imports = append(imports, imp.Path)
				}
			}
		}

		return imports
	}
	var result []string
	if values, ok := t.imports[path]; ok {
		for _, imp := range values {
			result = append(result, imp.Path)
		}
	}
	return result
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

func (t *DirTypes) matchDir(imports []*goImport, packageAlias string) *goImport {
	for _, cadndidate := range imports {
		if cadndidate.Name == packageAlias {
			return cadndidate
		}
	}
	suffix := "/" + packageAlias
	for _, cadndidate := range imports {
		if strings.HasSuffix(cadndidate.Path, suffix) {
			return cadndidate
		}
	}
	return nil
}
