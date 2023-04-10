package template

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/zoncoen/query-go"
	yamlextractor "github.com/zoncoen/query-go/extractor/yaml"

	"github.com/zoncoen/scenarigo/template/ast"
	"github.com/zoncoen/scenarigo/template/token"
)

func lookup(node ast.Node, data interface{}) (interface{}, error) {
	q, err := buildQuery(query.New(
		query.ExtractByStructTag("yaml", "json"),
		query.CustomExtractFunc(yamlextractor.MapSliceExtractFunc(false)),
	), node)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create query from AST")
	}
	v, err := q.Extract(defaultFuncs)
	if err != nil {
		v, err = q.Extract(data)
		if err != nil {
			return nil, err
		}
	}
	return Execute(v, data)
}

func buildQuery(q *query.Query, node ast.Node) (*query.Query, error) {
	var err error
	switch n := node.(type) {
	case *ast.Ident:
		return q.Key(n.Name), nil
	case *ast.SelectorExpr:
		q, err = buildQuery(q, n.X)
		if err != nil {
			return nil, err
		}
		return q.Key(n.Sel.Name), nil
	case *ast.IndexExpr:
		i, ok := n.Index.(*ast.BasicLit)
		if !ok || i.Kind != token.INT {
			return nil, errors.Errorf(`expected int but "%s"`, i.Kind.String())
		}
		idx, err := strconv.Atoi(i.Value)
		if err != nil {
			return nil, errors.Errorf(`expected int but "%s"`, i.Value)
		}
		q, err = buildQuery(q, n.X)
		if err != nil {
			return nil, err
		}
		return q.Index(idx), nil
	}
	return nil, errors.Errorf(`unknown node "%T"`, node)
}
