package ast

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"go/ast"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseExpr(t *testing.T) {

	rec := struct {
		ID   int
		Name string
		Sub  []*struct {
			Active bool
			Ts     *time.Time
		}
	}{}

	var testCases = []struct {
		description string
		expr        string
		expect      string
	}{
		{
			description: "compile inline type",
			expr:        fmt.Sprintf(`type Foo %v`, reflect.TypeOf(rec).String()),
			expect: `type Foo struct {
	ID   int
	Name string
	Sub  []*struct {
		Active bool
		Ts     *time.Time
	}
}`,
		},
	}

	for _, testCase := range testCases {
		root, err := ParseExpr(testCase.expr)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		fn, ok := root.(*ast.FuncLit)
		assert.True(t, ok)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		//toolbox.Dump(root)
		actual, err := StringifyExpr(fn.Body.List[0])
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		assert.EqualValues(t, testCase.expect, strings.TrimSpace(actual), testCase.description)
	}

}
