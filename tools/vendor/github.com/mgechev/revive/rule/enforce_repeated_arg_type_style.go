package rule

import (
	"fmt"
	"go/ast"
	"sync"

	"github.com/mgechev/revive/lint"
)

type enforceRepeatedArgTypeStyleType string

const (
	enforceRepeatedArgTypeStyleTypeAny   enforceRepeatedArgTypeStyleType = "any"
	enforceRepeatedArgTypeStyleTypeShort enforceRepeatedArgTypeStyleType = "short"
	enforceRepeatedArgTypeStyleTypeFull  enforceRepeatedArgTypeStyleType = "full"
)

func repeatedArgTypeStyleFromString(s string) enforceRepeatedArgTypeStyleType {
	switch s {
	case string(enforceRepeatedArgTypeStyleTypeAny), "":
		return enforceRepeatedArgTypeStyleTypeAny
	case string(enforceRepeatedArgTypeStyleTypeShort):
		return enforceRepeatedArgTypeStyleTypeShort
	case string(enforceRepeatedArgTypeStyleTypeFull):
		return enforceRepeatedArgTypeStyleTypeFull
	default:
		err := fmt.Errorf(
			"invalid repeated arg type style: %s (expecting one of %v)",
			s,
			[]enforceRepeatedArgTypeStyleType{
				enforceRepeatedArgTypeStyleTypeAny,
				enforceRepeatedArgTypeStyleTypeShort,
				enforceRepeatedArgTypeStyleTypeFull,
			},
		)

		panic(fmt.Sprintf("Invalid argument to the enforce-repeated-arg-type-style rule: %v", err))
	}
}

// EnforceRepeatedArgTypeStyleRule implements a rule to enforce repeated argument type style.
type EnforceRepeatedArgTypeStyleRule struct {
	funcArgStyle    enforceRepeatedArgTypeStyleType
	funcRetValStyle enforceRepeatedArgTypeStyleType

	configureOnce sync.Once
}

func (r *EnforceRepeatedArgTypeStyleRule) configure(arguments lint.Arguments) {
	r.funcArgStyle = enforceRepeatedArgTypeStyleTypeAny
	r.funcRetValStyle = enforceRepeatedArgTypeStyleTypeAny

	if len(arguments) == 0 {
		return
	}

	switch funcArgStyle := arguments[0].(type) {
	case string:
		r.funcArgStyle = repeatedArgTypeStyleFromString(funcArgStyle)
		r.funcRetValStyle = repeatedArgTypeStyleFromString(funcArgStyle)
	case map[string]any: // expecting map[string]string
		for k, v := range funcArgStyle {
			switch k {
			case "funcArgStyle":
				val, ok := v.(string)
				if !ok {
					panic(fmt.Sprintf("Invalid map value type for 'enforce-repeated-arg-type-style' rule. Expecting string, got %T", v))
				}
				r.funcArgStyle = repeatedArgTypeStyleFromString(val)
			case "funcRetValStyle":
				val, ok := v.(string)
				if !ok {
					panic(fmt.Sprintf("Invalid map value '%v' for 'enforce-repeated-arg-type-style' rule. Expecting string, got %T", v, v))
				}
				r.funcRetValStyle = repeatedArgTypeStyleFromString(val)
			default:
				panic(fmt.Sprintf("Invalid map key for 'enforce-repeated-arg-type-style' rule. Expecting 'funcArgStyle' or 'funcRetValStyle', got %v", k))
			}
		}
	default:
		panic(fmt.Sprintf("Invalid argument '%v' for 'import-alias-naming' rule. Expecting string or map[string]string, got %T", arguments[0], arguments[0]))
	}
}

// Apply applies the rule to a given file.
func (r *EnforceRepeatedArgTypeStyleRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configureOnce.Do(func() { r.configure(arguments) })

	if r.funcArgStyle == enforceRepeatedArgTypeStyleTypeAny && r.funcRetValStyle == enforceRepeatedArgTypeStyleTypeAny {
		// This linter is not configured, return no failures.
		return nil
	}

	var failures []lint.Failure

	astFile := file.AST
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if r.funcArgStyle == enforceRepeatedArgTypeStyleTypeFull {
				if fn.Type.Params != nil {
					for _, field := range fn.Type.Params.List {
						if len(field.Names) > 1 {
							failures = append(failures, lint.Failure{
								Confidence: 1,
								Node:       field,
								Category:   "style",
								Failure:    "argument types should not be omitted",
							})
						}
					}
				}
			}

			if r.funcArgStyle == enforceRepeatedArgTypeStyleTypeShort {
				var prevType ast.Expr
				if fn.Type.Params != nil {
					for _, field := range fn.Type.Params.List {
						prevTypeStr := gofmt(prevType)
						currentTypeStr := gofmt(field.Type)
						if currentTypeStr == prevTypeStr {
							failures = append(failures, lint.Failure{
								Confidence: 1,
								Node:       prevType,
								Category:   "style",
								Failure:    fmt.Sprintf("repeated argument type %q can be omitted", prevTypeStr),
							})
						}
						prevType = field.Type
					}
				}
			}

			if r.funcRetValStyle == enforceRepeatedArgTypeStyleTypeFull {
				if fn.Type.Results != nil {
					for _, field := range fn.Type.Results.List {
						if len(field.Names) > 1 {
							failures = append(failures, lint.Failure{
								Confidence: 1,
								Node:       field,
								Category:   "style",
								Failure:    "return types should not be omitted",
							})
						}
					}
				}
			}

			if r.funcRetValStyle == enforceRepeatedArgTypeStyleTypeShort {
				var prevType ast.Expr
				if fn.Type.Results != nil {
					for _, field := range fn.Type.Results.List {
						prevTypeStr := gofmt(prevType)
						currentTypeStr := gofmt(field.Type)
						if field.Names != nil && currentTypeStr == prevTypeStr {
							failures = append(failures, lint.Failure{
								Confidence: 1,
								Node:       prevType,
								Category:   "style",
								Failure:    fmt.Sprintf("repeated return type %q can be omitted", prevTypeStr),
							})
						}
						prevType = field.Type
					}
				}
			}
		}
		return true
	})

	return failures
}

// Name returns the name of the linter rule.
func (*EnforceRepeatedArgTypeStyleRule) Name() string {
	return "enforce-repeated-arg-type-style"
}
