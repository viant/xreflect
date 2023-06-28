package xreflect

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
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
