package xreflect

import (
	"go/ast"
	"go/parser"
)

//options represents parse dir option
type options struct {
	lookup    TypeLookupFn
	parseMode parser.Mode
	onField   func(typeName string, field *ast.Field) error
}

//Apply applies options
func (o *options) Apply(options ...Option) {
	if len(options) == 0 {
		return
	}
	for _, opt := range options {
		opt(o)
	}
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
