package xreflect

import (
	"go/ast"
	"go/parser"
)

//options represents parse dir option
type (
	parseOption struct {
		lookup    TypeLookupFn
		parseMode parser.Mode
		onField   func(typeName string, field *ast.Field) error
	}

	generateOption struct {
		packageName   string
		imports       []string
		snippetBefore string
		snippetAfter  string
	}
	options struct {
		parseOption
		generateOption
	}
)

//Apply applies options
func (o *options) Apply(options ...Option) {
	o.init()
	if len(options) == 0 {
		return
	}
	for _, opt := range options {
		opt(o)
	}

}

func (o *options) init() {
	o.packageName = "generated"
}

//Option represent parse option
type Option func(o *options)

//WithTypeLookupFn returns option with lookup fn
func WithTypeLookupFn(fn TypeLookupFn) Option {
	return func(o *options) {
		o.lookup = fn
	}
}

//WithParserMode return option to set parser mode i.r parser.ParseComments
func WithParserMode(mode parser.Mode) Option {
	return func(o *options) {
		o.parseMode = mode
	}
}

//WithOnField returns on field function
func WithOnField(fn func(typeName string, field *ast.Field) error) Option {
	return func(o *options) {
		o.onField = fn
	}
}

//WithPackage creates with package option
func WithPackage(pkg string) Option {
	return func(o *options) {
		o.packageName = pkg
	}
}

//WithImports creates import option
func WithImports(imports []string) Option {
	return func(o *options) {
		o.imports = imports
	}
}

//WithSnippetBefore creates snippet option
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
