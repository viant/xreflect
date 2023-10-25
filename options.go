package xreflect

import (
	"go/ast"
	"go/parser"
	"reflect"
)

// options represents parse dir option
type (
	parseOption struct {
		lookup    LookupType
		parseMode parser.Mode
		onField   func(typeName string, field *ast.Field) error
	}

	generateOption struct {
		imports       []string
		snippetBefore string
		snippetAfter  string
	}

	typeOptions struct {
		packagePath string
		definition  string
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

func WithReflectPackage(pkg string) Option {
	return func(o *options) {
		o.withReflectPackage = pkg
	}
}
