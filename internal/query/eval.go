package query

import (
	"errors"
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	"github.com/xunull/imfd/internal/media"
)

// Evaluator wraps a compiled expr program + the needles tied to it.
//
// Compile 一次 / Eval N 次（Section 4 perf 决议）。
type Evaluator struct {
	program *vm.Program
	needles []string
}

// SyntaxError indicates the filter expression failed to compile.
// list 命令应据此 exit 2，stderr 印 column number。
var SyntaxError = errors.New("filter syntax error")

// NewEvaluator compiles the filter expression.
//
// nil-safety 通过 envTypes 的 typed zero values 实现，不用 AST patch
// （expr type checker 走在 patch 之后，patch 路径走不通——see env.go）。
func NewEvaluator(expression string, needles []string) (*Evaluator, error) {
	if expression == "" {
		expression = "true"
	}

	// 构造 env 类型骨架；env 字段类型化 + needles 都是 string
	envSkeleton := envTypes()
	for i := range needles {
		envSkeleton[needleVar(i)] = ""
	}

	program, err := expr.Compile(expression,
		expr.Env(envSkeleton),
		expr.AsBool(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", SyntaxError, err)
	}
	return &Evaluator{program: program, needles: needles}, nil
}

// Match returns whether the record matches the filter expression.
func (e *Evaluator) Match(record *media.MediaRecord) (bool, error) {
	if e == nil || e.program == nil {
		return false, nil
	}
	env := BuildEnv(record, e.needles)
	out, err := expr.Run(e.program, env)
	if err != nil {
		return false, err
	}
	b, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("filter did not return bool, got %T", out)
	}
	return b, nil
}
