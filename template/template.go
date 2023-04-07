// Package template implements data-driven templates for generating a value.
package template

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
	"github.com/zoncoen/scenarigo/template/ast"
	"github.com/zoncoen/scenarigo/template/parser"
	"github.com/zoncoen/scenarigo/template/token"
)

// Template is the representation of a parsed template.
type Template struct {
	str  string
	expr ast.Expr

	executingLeftArrowExprArg bool
	argFuncs                  *funcStash
}

// New parses text as a template and returns it.
func New(str string) (*Template, error) {
	p := parser.NewParser(strings.NewReader(str))
	node, err := p.Parse()
	if err != nil {
		return nil, errors.Wrapf(err, `failed to parse "%s"`, str)
	}
	expr, ok := node.(ast.Expr)
	if !ok {
		return nil, errors.Errorf(`unknown node "%T"`, node)
	}
	return &Template{
		str:      str,
		expr:     expr,
		argFuncs: &funcStash{},
	}, nil
}

// Execute applies a parsed template to the specified data.
func (t *Template) Execute(data interface{}) (_ interface{}, retErr error) {
	defer func() {
		if err := recover(); err != nil {
			retErr = fmt.Errorf("failed to execute: panic: %s", err)
		}
	}()
	v, err := t.executeExpr(t.expr, data)
	if err != nil {
		if strings.Contains(t.str, "\n") {
			return nil, errors.Wrapf(err, "failed to execute: \n%s\n", t.str) //nolint:revive
		}
		return nil, errors.Wrapf(err, "failed to execute: %s", t.str)
	}
	return v, nil
}

func (t *Template) executeExpr(expr ast.Expr, data interface{}) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return t.executeBasicLit(e)
	case *ast.ParameterExpr:
		return t.executeParameterExpr(e, data)
	case *ast.BinaryExpr:
		return t.executeBinaryExpr(e, data)
	case *ast.Ident:
		return lookup(e, data)
	case *ast.SelectorExpr:
		return lookup(e, data)
	case *ast.IndexExpr:
		return lookup(e, data)
	case *ast.CallExpr:
		return t.executeFuncCall(e, data)
	case *ast.LeftArrowExpr:
		return t.executeLeftArrowExpr(e, data)
	default:
		return nil, errors.Errorf(`unknown expression "%T"`, e)
	}
}

func (t *Template) executeBasicLit(lit *ast.BasicLit) (interface{}, error) {
	switch lit.Kind {
	case token.STRING:
		return lit.Value, nil
	case token.INT:
		i, err := strconv.Atoi(lit.Value)
		if err != nil {
			return nil, errors.Wrapf(err, `invalid AST: "%s" is not an integer`, lit.Value)
		}
		return i, nil
	case token.FLOAT:
		f, err := strconv.ParseFloat(lit.Value, 64)
		if err != nil {
			return nil, errors.Wrapf(err, `invalid AST: "%s" is not a float`, lit.Value)
		}
		return f, nil
	case token.BOOL:
		switch lit.Value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return nil, errors.Errorf(`invalid bool literal "%s"`, lit.Value)
		}
	default:
		return nil, errors.Errorf(`unknown basic literal "%s"`, lit.Kind.String())
	}
}

func (t *Template) executeParameterExpr(e *ast.ParameterExpr, data interface{}) (interface{}, error) {
	if e.X == nil {
		return "", nil
	}
	v, err := t.executeExpr(e.X, data)
	if err != nil {
		return nil, err
	}
	if t.executingLeftArrowExprArg {
		// HACK: Left arrow function requires its argument is YAML string to unmarshal into the Go value.
		// Replace the function into a string temporary.
		// It will be restored in UnmarshalArg method.
		if reflectutil.Elem(reflect.ValueOf(v)).Kind() == reflect.Func {
			name := t.argFuncs.save(v)
			if e.Quoted {
				return fmt.Sprintf("'{{%s}}'", name), nil
			}
			return fmt.Sprintf("{{%s}}", name), nil
		}

		// Left arrow function arguments must be a string in YAML.
		b, err := yaml.Marshal(v)
		if err != nil {
			return nil, err
		}
		return strings.TrimSuffix(string(b), "\n"), nil
	}
	return v, nil
}

func (t *Template) executeBinaryExpr(e *ast.BinaryExpr, data interface{}) (interface{}, error) {
	x, err := t.executeExpr(e.X, data)
	if err != nil {
		return nil, err
	}
	y, err := t.executeExpr(e.Y, data)
	if err != nil {
		return nil, err
	}
	var withIndent bool
	if t.executingLeftArrowExprArg {
		if _, ok := (e.Y).(*ast.ParameterExpr); ok {
			withIndent = true
		}
	}
	switch e.Op {
	case token.ADD:
		return t.add(x, y, withIndent)
	default:
		return nil, errors.Errorf(`unknown operation "%s"`, e.Op.String())
	}
}

func (t *Template) add(x, y interface{}, withIndent bool) (interface{}, error) {
	strX, err := t.stringize(x)
	if err != nil {
		return nil, errors.Wrap(err, "failed to concat strings")
	}
	strY, err := t.stringize(y)
	if err != nil {
		return nil, errors.Wrap(err, "failed to concat strings")
	}
	if withIndent {
		strY, err = t.addIndent(strY, strX)
		if err != nil {
			return nil, errors.Wrap(err, "failed to concat strings")
		}
	}
	return strX + strY, nil
}

// align indents of marshaled texts
//
//	example: addIndent("a: 1\nb:2", "- ")
//	=== before ===
//	- a: 1
//	b: 2
//	=== after ===
//	- a: 1
//	  b: 2
func (t *Template) addIndent(str, preStr string) (string, error) {
	if t.executingLeftArrowExprArg {
		if strings.ContainsRune(str, '\n') && preStr != "" {
			lines := strings.Split(preStr, "\n")
			prefix := strings.Repeat(" ", len([]rune(lines[len(lines)-1])))
			var b strings.Builder
			for i, s := range strings.Split(str, "\n") {
				if i != 0 {
					if _, err := b.WriteRune('\n'); err != nil {
						return "", err
					}
					if s != "" {
						if _, err := b.WriteString(prefix); err != nil {
							return "", err
						}
					}
				}
				if _, err := b.WriteString(s); err != nil {
					return "", err
				}
			}
			return b.String(), nil
		}
	}
	return str, nil
}

func (t *Template) stringize(v interface{}) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", errors.Errorf("expect string but got %T", v)
	}
	return s, nil
}

func (t *Template) requiredFuncArgType(funcType reflect.Type, argIdx int) reflect.Type {
	if !funcType.IsVariadic() {
		return funcType.In(argIdx)
	}

	argNum := funcType.NumIn()
	lastArgIdx := argNum - 1
	if argIdx < lastArgIdx {
		return funcType.In(argIdx)
	}

	return funcType.In(lastArgIdx).Elem()
}

func (t *Template) executeFuncCall(call *ast.CallExpr, data interface{}) (interface{}, error) {
	var fn reflect.Value
	fnName := "function"
	args := make([]reflect.Value, 0, len(call.Args)+1)
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if ok {
		x, err := t.executeExpr(selector.X, data)
		if err != nil {
			return nil, err
		}
		v, err := lookup(selector.Sel, x)
		if err == nil {
			fn = reflect.ValueOf(v)
		} else {
			r, m, ok := getMethod(x, selector.Sel.Name)
			if !ok {
				return nil, err
			}
			fn = m.Func
			args = append(args, r)
		}
		fnName = selector.Sel.Name
	} else {
		f, err := t.executeExpr(call.Fun, data)
		if err != nil {
			return nil, err
		}
		fn = reflect.ValueOf(f)
		if id, ok := call.Fun.(*ast.Ident); ok {
			fnName = id.Name
		}
	}
	if fn.Kind() != reflect.Func {
		return nil, errors.Errorf("not function")
	}
	fnType := fn.Type()
	argNum := len(args) + len(call.Args)
	if fnType.IsVariadic() {
		minArgNum := fnType.NumIn() - 1
		if argNum < minArgNum {
			return nil, errors.Errorf(
				"too few arguments to function: expected minimum argument number is %d. but specified %d arguments",
				minArgNum, argNum,
			)
		}
	} else if fnType.NumIn() != argNum {
		return nil, errors.Errorf(
			"expected function argument number is %d but specified %d arguments",
			fn.Type().NumIn(), argNum,
		)
	}

	args, err := t.executeArgs(fnName, fnType, args, call.Args, data)
	if err != nil {
		return nil, err
	}

	vs := fn.Call(args)
	switch len(vs) {
	case 1:
		if !vs[0].IsValid() || !vs[0].CanInterface() {
			return nil, errors.Errorf("function returns an invalid value")
		}
		return vs[0].Interface(), nil
	case 2:
		if !vs[0].IsValid() || !vs[0].CanInterface() {
			return nil, errors.Errorf("first reruned value is invalid")
		}
		if !vs[1].IsValid() || !vs[1].CanInterface() {
			return nil, errors.Errorf("second reruned value is invalid")
		}
		if vs[1].Type() != reflectutil.TypeError {
			return nil, errors.Errorf("second returned value must be an error")
		}
		if !vs[1].IsNil() {
			return nil, vs[1].Interface().(error) //nolint:forcetypeassert
		}
		return vs[0].Interface(), nil
	default:
		return nil, errors.Errorf("function should return a value or a value and an error")
	}
}

func getMethod(in interface{}, name string) (reflect.Value, *reflect.Method, bool) {
	r := reflectutil.Elem(reflect.ValueOf(in))
	m, ok := r.Type().MethodByName(name)
	if ok {
		return r, &m, true
	}
	if r.CanAddr() {
		r = r.Addr()
		m, ok := r.Type().MethodByName(name)
		if ok {
			return r, &m, true
		}
	} else {
		ptr := makePtr(r)
		m, ok := ptr.Type().MethodByName(name)
		if ok {
			return ptr, &m, true
		}
	}
	return reflect.Value{}, nil, false
}

func (t *Template) executeArgs(fnName string, fnType reflect.Type, vs []reflect.Value, args []ast.Expr, data interface{}) ([]reflect.Value, error) {
	for i, arg := range args {
		a, err := t.executeExpr(arg, data)
		if err != nil {
			return nil, err
		}
		requiredType := t.requiredFuncArgType(fnType, len(vs))
		v := reflect.ValueOf(a)
		if v.IsValid() {
			vv, ok, _ := reflectutil.Convert(requiredType, v)
			if ok {
				v = vv
			}
		}
		if typ := v.Type(); typ != requiredType {
			return nil, errors.Errorf("can't use %s as %s in arguments[%d] to %s", typ, requiredType, i, fnName)
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func (t *Template) executeLeftArrowExpr(e *ast.LeftArrowExpr, data interface{}) (interface{}, error) {
	v, err := t.executeExpr(e.Fun, data)
	if err != nil {
		return nil, err
	}
	f, ok := v.(Func)
	if !ok {
		return nil, errors.Errorf(`expect template function but got %T`, e)
	}

	// without arg in map key
	if e.Arg == nil {
		return &lazyFunc{f}, nil
	}

	v, err = t.executeLeftArrowExprArg(e.Arg, data)
	if err != nil {
		return nil, err
	}
	argStr, ok := v.(string)
	if !ok {
		return nil, errors.Errorf(`expect string but got %T`, v)
	}
	arg, err := f.UnmarshalArg(func(v interface{}) error {
		if err := yaml.NewDecoder(strings.NewReader(argStr), yaml.UseOrderedMap(), yaml.Strict()).Decode(v); err != nil {
			return err
		}
		// Restore functions that are replaced into strings.
		// See the "HACK" comment of *Template.executeParameterExpr method.
		arg, err := Execute(v, t.argFuncs)
		if err != nil {
			return err
		}
		// NOTE: Decode method ensures that v is a pointer.
		rv := reflect.ValueOf(v).Elem()
		ev, err := convert(rv.Type())(reflect.ValueOf(arg), nil)
		if err != nil {
			return err
		}
		rv.Set(ev)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return f.Exec(arg)
}

func (t *Template) executeLeftArrowExprArg(arg ast.Expr, data interface{}) (interface{}, error) {
	tt := &Template{
		expr:                      arg,
		executingLeftArrowExprArg: true,
		argFuncs:                  t.argFuncs,
	}
	v, err := tt.Execute(data)
	return v, err
}

type funcStash map[string]interface{}

func (s *funcStash) save(f interface{}) string {
	if *s == nil {
		*s = funcStash{}
	}
	name := fmt.Sprintf("func-%d", len(*s))
	(*s)[name] = f
	return name
}

// Func represents a left arrow function.
type Func interface {
	Exec(arg interface{}) (interface{}, error)
	UnmarshalArg(unmarshal func(interface{}) error) (interface{}, error)
}

type lazyFunc struct {
	f Func
}
