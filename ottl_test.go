package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type scenario struct {
	Name        string
	Function    string
	Tests       []ScenarioTest
	target      ottl.GetSetter[pcommon.Value]
	Pattern     string
	Replacement string
	optFunction ottl.Optional[ottl.FunctionGetter[pcommon.Value]]
	want        func(pcommon.Value)
}

type ScenarioTest struct {
	Input  string
	Expect string
}

func TestReplacements(t *testing.T) {
	stdFunctions := ottlfuncs.StandardFuncs[pcommon.Value]()
	target := &ottl.StandardGetSetter[pcommon.Value]{
		Getter: func(ctx context.Context, tCtx pcommon.Value) (any, error) {
			return tCtx.Str(), nil
		},
		Setter: func(ctx context.Context, tCtx pcommon.Value, val any) error {
			tCtx.SetStr(val.(string))
			return nil
		},
	}
	content, err := os.ReadFile("./tests.json")
	if err != nil {
		t.Fatal("cannot read test file")
	}
	var scenarios []scenario
	err = json.Unmarshal(content, &scenarios)
	if err != nil {
		t.Fatal("cannot unmarshal tests")
	}

	for _, tt := range scenarios {
		for _, test := range tt.Tests {
			t.Run(fmt.Sprintf("%s/%s", tt.Name, test.Expect), func(t *testing.T) {
				scenarioValue := pcommon.NewValueStr(test.Input)
				replacePatternFunctionFactory := stdFunctions[tt.Function]
				assert.Equal(t, tt.Function, replacePatternFunctionFactory.Name())
				args := replacePatternFunctionFactory.CreateDefaultArguments()
				argsReal, ok := args.(*ottlfuncs.ReplacePatternArguments[pcommon.Value])
				if !ok {
					t.Fatal("not ok")
				}
				argsReal.Target = target
				argsReal.RegexPattern = tt.Pattern
				argsReal.Replacement = ottl.StandardStringGetter[pcommon.Value]{
					Getter: func(context.Context, pcommon.Value) (any, error) {
						return tt.Replacement, nil
					},
				}
				argsReal.Function = tt.optFunction
				fctx := ottl.FunctionContext{
					Set: componenttest.NewNopTelemetrySettings(),
				}

				exprFunc, err := replacePatternFunctionFactory.CreateFunction(fctx, argsReal)
				assert.NoError(t, err)

				result, err := exprFunc(nil, scenarioValue)
				assert.NoError(t, err)
				assert.Nil(t, result)

				expected := pcommon.NewValueStr(test.Expect)
				assert.Equal(t, expected, scenarioValue)
			})
		}
	}
}
