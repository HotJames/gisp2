package gisp

import (
	"fmt"

	p "github.com/Dwarfartisan/goparsec2"
)

// Let 实现 let 环境
type Let struct {
	Meta    map[string]interface{}
	Content List
}

// LetFunc 构造一个 Let 环境
func LetFunc(env Env, args ...interface{}) (Lisp, error) {
	st := p.NewBasicState(args)
	_, err := TypeAs(LIST)(&st)
	if err != nil {
		return nil, fmt.Errorf("Let Args Error: expect args list but error: %v", err)
	}

	local := map[string]Var{}
	vars := args[0].(List)
	for _, v := range vars {
		declares := v.(List)
		varb := declares[0].(Atom)
		slot := VarSlot(varb.Type)
		value, err := Eval(env, (declares[1]))
		if err != nil {
			return nil, err
		}
		slot.Set(value)
		local[varb.Name] = slot
	}
	meta := map[string]interface{}{
		"local": local,
	}
	let := Let{meta, args}
	return let, nil
}

// LetExpr 将 let => (let ((a, value), (b, value)...) ...) 形式构造为一个 let 环境
func LetExpr(env Env, args ...interface{}) (Tasker, error) {
	var (
		vars List
		ok   bool
	)
	if len(args) < 1 {
		return nil, fmt.Errorf("let args error: expect vars list at last but a empty let as (let )")
	}
	if vars, ok = args[0].(List); !ok {
		return nil, fmt.Errorf("let args error: expect vars list but %v", args[0])
	}
	return func(env Env) (interface{}, error) {
		local := map[string]Var{}
		for _, v := range vars {
			declares := v.(List)
			varb := declares[0].(Atom)
			slot := VarSlot(varb.Type)
			value, err := Eval(env, (declares[1]))
			if err != nil {
				return nil, err
			}
			slot.Set(value)
			local[varb.Name] = slot
		}
		meta := map[string]interface{}{
			"local": local,
		}
		let := Let{meta, args[1:]}
		return let.Eval(env)
	}, nil
}

// Defvar 实现 Env.Defvar
func (let Let) Defvar(name string, slot Var) error {
	if _, ok := let.Local(name); ok {
		return fmt.Errorf("local name %s is exists", name)
	}
	local := let.Meta["local"].(map[string]Var)
	local[name] = slot
	return nil
}

// Defun 实现 Env.Defun
func (let Let) Defun(name string, functor Functor) error {
	if s, ok := let.Local(name); ok {
		switch slot := s.(type) {
		case Func:
			slot.Overload(functor)
		case Var:
			return fmt.Errorf("%s defined as a var", name)
		default:
			return fmt.Errorf("exists name %s isn't Expr", name)
		}
	}
	local := let.Meta["local"].(map[string]interface{})
	local[name] = NewFunction(name, let, functor)
	return nil
}

// Setvar 实现 Env.Setvar
func (let Let) Setvar(name string, value interface{}) error {
	if _, ok := let.Local(name); ok {
		local := let.Meta["local"].(map[string]Var)
		local[name].Set(value)
		return nil
	}
	global := let.Meta["global"].(Env)
	return global.Setvar(name, value)
}

// Local 实现 Env.Local
func (let Let) Local(name string) (interface{}, bool) {
	local := let.Meta["local"].(map[string]Var)
	if slot, ok := local[name]; ok {
		return slot.Get(), true
	}
	return nil, false
}

// Lookup 实现 Env.Lookup
func (let Let) Lookup(name string) (interface{}, bool) {
	if value, ok := let.Local(name); ok {
		return value, true
	}
	return let.Global(name)

}

// Global 实现 Env.Global
func (let Let) Global(name string) (interface{}, bool) {
	global := let.Meta["global"].(Env)
	return global.Lookup(name)
}

// Eval 实现 Lisp.Eval
func (let Let) Eval(env Env) (interface{}, error) {
	let.Meta["global"] = env
	l := len(let.Content)
	switch l {
	case 0:
		return nil, nil
	case 1:
		return Eval(let, let.Content[0])
	default:
		for _, Expr := range let.Content[:l-1] {
			_, err := Eval(let, Expr)
			if err != nil {
				return nil, err
			}
		}
		Expr := let.Content[l-1]
		return Eval(let, Expr)
	}
}
