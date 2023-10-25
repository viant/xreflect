package xreflect

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestStringify(t *testing.T) {
	testCases := []struct {
		description string
		rType       reflect.Type
		tag         string
		expected    string
	}{
		{
			description: "slice of autogen type",
			rType: reflect.SliceOf(reflect.PtrTo(reflect.StructOf([]reflect.StructField{
				{
					Name: "ID",
					Type: IntType,
				},
				{
					Name: "Name",
					Type: StringType,
				},
			}))),
			expected: "[]*Foo",
			tag:      fmt.Sprintf(`%v:"Foo"`, TagTypeName),
		},
	}

	for _, testCase := range testCases {
		result := Stringify(testCase.rType, reflect.StructTag(testCase.tag))
		assert.Equal(t, testCase.expected, result, testCase.description)
	}
}

func TestType_Body(t *testing.T) {

	type Bar struct {
		Id     int
		Active bool
	}

	type Test struct {
		secret string
		Name   string
	}

	var foo = struct {
		Time *time.Time `format:"tz=utc"`

		Id   int `json:"Id"`
		Test struct {
			secret string
			Name   string
		} `typeName:"Test"`
		Inline struct {
			secret string
			Name   string
		}
		Bar Bar
	}{}

	var testCases = []struct {
		description string
		Type        *Type
		Dep         []*Type
		expect      string
	}{
		{
			description: "inlined mixed type",
			Dep: []*Type{
				NewType("Bar", WithReflectType(reflect.TypeOf(Bar{})), WithPackage("xreflect")),
				NewType("Test", WithReflectType(reflect.TypeOf(Test{})), WithPackage("xreflect")),
			},
			Type:   NewType("Foo", WithReflectType(reflect.TypeOf(foo)), WithPackage("xreflect")),
			expect: `struct{Time *time.Time "format:\"tz=utc\""; Id int "json:\"Id\""; Test Test "eName:\"Test\""; Inline struct{secret string; Name string; }; Bar Bar; }`,
		},
	}
	for _, testCase := range testCases {
		actual := testCase.Type.Body()
		assert.Equal(t, testCase.expect, actual, testCase.description)

		types := NewTypes()
		for _, dep := range testCase.Dep {
			err := types.Register(dep.Name, WithPackage(dep.Package), WithReflectType(dep.Type))
			if !assert.Nil(t, err, testCase.description) {
				return
			}
		}
		err := types.Register(testCase.Type.Name, WithPackage(testCase.Type.Package), WithTypeDefinition(actual))
		if !assert.Nil(t, err, testCase.description) {
			return
		}
		rType, err := types.Lookup(testCase.Type.Name, WithPackage("xreflect"))
		if !assert.Nil(t, err, testCase.description) {
			return
		}
		assert.NotNil(t, rType)
	}
}
