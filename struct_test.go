package xreflect

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestGenerateGoStruct(t *testing.T) {
	type Bar struct {
		ID   int
		Name string
	}

	type Foo struct {
		ID   *int
		Name *string
		Time *time.Time
	}

	testcases := []struct {
		description string
		rType       reflect.Type
		name        string
		expected    string
	}{
		{
			description: "primitive types",
			rType:       IntType,
			name:        "Foo",
			expected: `package generated

type Foo int
`,
		},
		{
			description: "primitive ptr",
			rType:       reflect.PtrTo(IntType),
			name:        "Foo",
			expected: `package generated

type Foo *int
`,
		},
		{
			description: "generated struct",
			rType: reflect.StructOf([]reflect.StructField{
				{
					Name: "Id",
					Type: IntType,
				},
				{
					Name: "Name",
					Type: StringType,
				},
				{
					Name: "Active",
					Type: BoolType,
				},
			}),
			name: "Foo",
			expected: `package generated

type Foo struct {
	Id     int
	Name   string
	Active bool
}
`,
		},
		{
			description: "nested structs",
			rType: reflect.StructOf([]reflect.StructField{
				{
					Name: "Id",
					Type: IntType,
				},
				{
					Name: "Name",
					Type: StringType,
				},
				{
					Name: "Bar",
					Type: reflect.StructOf([]reflect.StructField{
						{
							Name: "BarId",
							Type: IntType,
						},
						{
							Name: "Price",
							Type: Float64Type,
						},
					}),
				},
			}),
			name: "Foo",
			expected: `package generated

type Foo struct {
	Id   int
	Name string
	Bar  Bar
}

type Bar struct {
	BarId int
	Price float64
}
`,
		},
		{
			description: "tags",
			rType: reflect.StructOf([]reflect.StructField{
				{
					Name: "Id",
					Type: IntType,
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Name",
					Type: StringType,
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Bar",
					Type: reflect.StructOf([]reflect.StructField{
						{
							Name: "BarId",
							Type: IntType,
						},
						{
							Name: "Price",
							Type: Float64Type,
						},
					}),
				},
			}),
			name:     "Foo",
			expected: "package generated\n\ntype Foo struct {\n\tId   int    `json:\",omitempty\"`\n\tName string `json:\",omitempty\"`\n\tBar  Bar\n}\n\ntype Bar struct {\n\tBarId int\n\tPrice float64\n}\n",
		},
		{
			description: "golang types",
			rType: reflect.StructOf([]reflect.StructField{
				{
					Name: "Id",
					Type: IntType,
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Name",
					Type: StringType,
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Bar",
					Type: reflect.TypeOf(Bar{}),
				},
			}),
			name:     "Foo",
			expected: "package generated\n\nimport (\n\t\"github.com/viant/xreflect\"\n)\n\ntype Foo struct {\n\tId   int    `json:\",omitempty\"`\n\tName string `json:\",omitempty\"`\n\tBar  xreflect.Bar\n}\n",
		},
		{
			description: "type renamed",
			rType: reflect.StructOf([]reflect.StructField{
				{
					Name: "Id",
					Type: IntType,
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Name",
					Type: StringType,
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Bar",
					Tag:  reflect.StructTag(fmt.Sprintf(`%v:"%v"`, TagTypeName, "BarType")),
					Type: reflect.PtrTo(reflect.StructOf([]reflect.StructField{
						{
							Name: "BarName",
							Type: StringType,
						},
						{
							Name: "BarID",
							Type: Int64Type,
						},
					})),
				},
			}),
			name:     "Foo",
			expected: "package generated\n\ntype Foo struct {\n\tId   int      `json:\",omitempty\"`\n\tName string   `json:\",omitempty\"`\n\tBar  *BarType `typeName:\"BarType\"`\n}\n\ntype BarType struct {\n\tBarName string\n\tBarID   int64\n}\n",
		},
		{
			description: "time.Time",
			rType:       reflect.TypeOf(Foo{}),
			name:        "Foo",
			expected: `package generated

import (
	"time"
)

type Foo struct {
	ID   *int
	Name *string
	Time *time.Time
}
`,
		},
	}

	//for _, testCase := range testcases[len(testcases)-1:] {
	for _, testCase := range testcases {
		goStruct := GenerateStruct(testCase.name, testCase.rType)
		if !assert.Equal(t, testCase.expected, goStruct, testCase.description) {
			fmt.Printf("")
		}
	}
}
