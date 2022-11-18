package xreflect

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestGenerateGoStruct(t *testing.T) {
	type Bar struct {
		ID   int
		Name string
	}

	testcases := []struct {
		description string
		rType       reflect.Type
		name        string
		expected    string
	}{
		{
			description: "primitive types",
			rType:       reflect.TypeOf(0),
			name:        "Foo",
			expected: `package generated

type Foo int
`,
		},
		{
			description: "primitive ptr",
			rType:       reflect.PtrTo(reflect.TypeOf(0)),
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
					Type: reflect.TypeOf(0),
				},
				{
					Name: "Name",
					Type: reflect.TypeOf(""),
				},
				{
					Name: "Active",
					Type: reflect.TypeOf(false),
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
					Type: reflect.TypeOf(0),
				},
				{
					Name: "Name",
					Type: reflect.TypeOf(""),
				},
				{
					Name: "Bar",
					Type: reflect.StructOf([]reflect.StructField{
						{
							Name: "BarId",
							Type: reflect.TypeOf(0),
						},
						{
							Name: "Price",
							Type: reflect.TypeOf(0.0),
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
					Type: reflect.TypeOf(0),
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Name",
					Type: reflect.TypeOf(""),
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Bar",
					Type: reflect.StructOf([]reflect.StructField{
						{
							Name: "BarId",
							Type: reflect.TypeOf(0),
						},
						{
							Name: "Price",
							Type: reflect.TypeOf(0.0),
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
					Type: reflect.TypeOf(0),
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Name",
					Type: reflect.TypeOf(""),
					Tag:  "json:\",omitempty\"",
				},
				{
					Name: "Bar",
					Type: reflect.TypeOf(Bar{}),
				},
			}),
			name:     "Foo",
			expected: "package generated\n\nimport (\n\t\"github.com/viant/xreflect\"\n)\n\ntype Foo struct {\n\tId   int    `json:\",omitempty\"`\n\tName string `json:\",omitempty\"`\n\tBar  Bar\n}\n",
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
