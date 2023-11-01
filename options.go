package xreflect

import (
	"go/ast"
	"go/parser"
	"golang.org/x/mod/modfile"
	"reflect"
)

// options represents parse dir option
type (
	parseOption struct {
		lookup         LookupType
		parseMode      parser.Mode
		module         *modfile.Module
		moduleLocation string
		onField        func(typeName string, field *ast.Field) error
		onStruct       func(spec *ast.TypeSpec, aStruct *ast.StructType) error
		onLookup       func(packagePath, pkg, typeName string, rType reflect.Type)
	}

	generateOption struct {
		imports       []string
		snippetBefore string
		snippetAfter  string
		packageTypes  []*Type
		importModule  map[string]string
	}

	registryOptions struct {
		withTypes          []*Type
		withReflectTypes   []reflect.Type
		withReflectPackage string
	}

	options struct {
		parseOption
		generateOption
		registryOptions
		Type
	}
)

func (o *generateOption) getPackageType(name string) *Type {
	if len(o.packageTypes) == 0 {
		return nil
	}
	for _, candidate := range o.packageTypes {
		if candidate.Name == name {
			return candidate
		}
	}
	return nil
}

// Apply applies options
func (o *options) Apply(options ...Option) {
	o.init()
	if len(options) == 0 {
		return
	}
	for _, opt := range options {
		opt(o)
	}

}

func (o *options) init() {}

func (o *options) initGen() {
	if o.Package == "" {
		o.Package = "generated"
	}
}

// Option represent parse option
type Option func(o *options)

// WithTypeLookup returns option with lookup fn
func WithTypeLookup(fn LookupType) Option {
	return func(o *options) {
		o.lookup = fn
	}
}

// WithParserMode return option to set parser mode i.r parser.ParseComments
func WithParserMode(mode parser.Mode) Option {
	return func(o *options) {
		o.parseMode = mode
	}
}

// WithOnField returns on field function
func WithOnField(fn func(typeName string, field *ast.Field) error) Option {
	return func(o *options) {
		o.onField = fn
	}
}

// WithPackage creates with package option
func WithPackage(pkg string) Option {
	return func(o *options) {
		o.Package = pkg

	}
}

// WithImports creates import option
func WithImports(imports []string) Option {
	return func(o *options) {
		o.imports = imports
	}
}

// WithSnippetBefore creates snippet option
func WithSnippetBefore(snippet string) Option {
	return func(o *options) {
		o.snippetBefore = snippet
	}
}

func WithSnippetAfter(snippet string) Option {
	return func(o *options) {
		o.snippetAfter = snippet
	}
}

func WithPackagePath(pkgPath string) Option {
	return func(t *options) {
		t.PackagePath = pkgPath
	}
}

func WithTypeDefinition(definition string) Option {
	return func(t *options) {
		t.Definition = definition
	}
}

func WithRegistry(r *Types) Option {
	return func(o *options) {
		o.Registry = r
	}
}

// WithReflectType update Type with reflect.Type
func WithReflectType(rType reflect.Type) Option {
	return func(t *options) {
		t.Type.Type = rType
	}
}

func WithReflectTypes(types ...reflect.Type) Option {
	return func(o *options) {
		o.withReflectTypes = types
	}
}

func WithTypes(types ...*Type) Option {
	return func(o *options) {
		o.withTypes = types
	}
}

func WithImportModule(importModule map[string]string) Option {
	return func(o *options) {
		o.importModule = importModule
	}
}

func WithReflectPackage(pkg string) Option {
	return func(o *options) {
		o.withReflectPackage = pkg
	}
}

// WithPackageTypes return option with package types
func WithPackageTypes(pkgTypes ...*Type) Option {
	return func(o *options) {
		o.packageTypes = pkgTypes
	}
}

// WithModule return option with module
func WithModule(module *modfile.Module, location string) Option {
	return func(o *options) {
		o.module = module
		o.moduleLocation = location
	}
}

// WithOnLookup return on lookup notifier option
func WithOnLookup(fn func(packagePath, pkg, typeName string, rType reflect.Type)) Option {
	return func(o *options) {
		o.onLookup = fn
	}
}

// WithOnLookup return on lookup notifier option
func WithOnStruct(fn func(spec *ast.TypeSpec, aStruct *ast.StructType) error) Option {
	return func(o *options) {
		o.onStruct = fn
	}
}

func withOptions(opt *options) Option {
	return func(o *options) {
		*o = *opt
	}
}
