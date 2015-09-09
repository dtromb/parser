package relex

import (
	"fmt"
	"testing"
	"github.com/dtromb/parser"
	"github.com/dtromb/parser/earley"
	"math/big"
)

var SAMPLE_PROGRAM string = `
	n = 15
	a = 1+n^2*2
	a = a*n/10
	print a
	print n^2
`

type Context struct {
	vars map[string]*big.Int
}

type Statement interface {
	Execute(ctx *Context)
}

type Expression interface {
	Evaluate(ctx *Context) *big.Int
}

type StatementList struct {
	statements []Statement
}

type Assignment struct {
	varname string
	value Expression
}
func (a *Assignment) Execute(ctx *Context) {
	ctx.vars[a.varname] = a.value.Evaluate(ctx)
}

type Output struct {
	value Expression
}
func (o *Output) Execute(ctx *Context) {
	fmt.Printf("%s\n",o.value.Evaluate(ctx).String())
}

type VariableExpression struct {
	varname string
}
func (ve *VariableExpression) Evaluate(ctx *Context) *big.Int {
	if val, has := ctx.vars[ve.varname]; has {
		return val
	}
	panic("unknown variable '"+ve.varname+"' evaluated")
}

type Operator int
const (
	ADDITION	Operator = iota
	SUBTRACTION
	MULTIPLICATION
	DIVISION
	MODULUS
	EXPONENTIATION
)
type BinaryOperation struct {
	left Expression
	right Expression
	op Operator
}
func (bo *BinaryOperation) Evaluate(ctx *Context) *big.Int {
	l := bo.left.Evaluate(ctx)
	r := bo.right.Evaluate(ctx)
	z := big.NewInt(0)
	switch(bo.op) {
		case ADDITION: {
			z.Add(l,r)
		}
		case SUBTRACTION: {
			z.Sub(l,r)
		}
		case MULTIPLICATION: {
			z.Mul(l,r)
		}
		case DIVISION: {
			z.Div(l,r)
		}
		case MODULUS: {
			z.Mod(l, r)
		}
		case EXPONENTIATION: {
			z.Exp(l, r, nil)
		}
	}
	return z
}

type Negation struct {
	arg Expression
}
func (n *Negation) Evaluate(ctx *Context) *big.Int {
	z := n.arg.Evaluate(ctx)
	return z.Neg(z)
}


type Literal struct {
	val *big.Int
}
func (l *Literal) Evaluate(ctx *Context) *big.Int {
	return l.val
}

func binaryOpCtor(p parser.Production, values []interface{}) (interface{},error) {
	return &BinaryOperation{left: values[0].(Expression),
							right: values[2].(Expression),
							op: values[1].(Operator)}, nil
}
				
func TestRelex(t *testing.T) {
	
	gb := parser.OpenGrammarBuilder()
	gb.Name("simple-calculator"). 
		Terminals("ID","EQ","PRINT","POW","PLUS","MINUS","TIMES","DIV","MOD","LP","RP","NUM"). 
		Nonterminals("program","statement","assignment","output","expr","aopfree","aop",
		             "mopfree","mop","unit"). 
		Rule().Lhs("`*").Rhs("program","`.").
		Rule().Lhs("program").Rhs("statement").
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				return &StatementList{statements: []Statement{values[0].(Statement)}}, nil
			}).
		Rule().Lhs("program").Rhs("statement","program"). 
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				slist := values[1].(*StatementList)
				slist.statements = append(slist.statements, values[0].(Statement))
				return slist, nil
			}).
		Rule().Lhs("statement").Rhs("assignment").
		Rule().Lhs("statement").Rhs("output"). 
		Rule().Lhs("assignment").Rhs("ID", "EQ", "expr").
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				return &Assignment{varname: values[0].(parser.Stringable).String(), 
								   value: values[2].(Expression)}, nil
			}).
		Rule().Lhs("output").Rhs("PRINT","expr"). 
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				return &Output{value: values[1].(Expression)}, nil
			}).
		Rule().Lhs("expr").Rhs("aopfree"). 
		Rule().Lhs("expr").Rhs("expr","aop","aopfree").Value(binaryOpCtor).
		Rule().Lhs("aopfree").Rhs("aopfree","mop","mopfree").Value(binaryOpCtor). 
		Rule().Lhs("mopfree").Rhs("mopfree","POW","unit").Value(binaryOpCtor).
		Rule().Lhs("aop").Rhs("PLUS"). 
		Rule().Lhs("aop").Rhs("MINUS").
		Rule().Lhs("mop").Rhs("TIMES"). 
		Rule().Lhs("mop").Rhs("DIV"). 
		Rule().Lhs("mop").Rhs("MOD"). 
		Rule().Lhs("unit").Rhs("ID").
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				return &VariableExpression{varname: values[0].(parser.Stringable).String()}, nil
			}).
		Rule().Lhs("unit").Rhs("MINUS", "unit").
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				return &Negation{arg: values[1].(Expression)}, nil
			}).
		Rule().Lhs("unit").Rhs("LP", "expr", "RP"). 
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				return values[1], nil
			}).
		Rule().Lhs("unit").Rhs("NUM").
			Value(func(p parser.Production, values []interface{}) (interface{},error) {
				return &Literal{val: values[0].(*big.Int)}, nil
			})
	g, err := gb.Build()
	if err != nil {
		t.Error(err)
		return
	}
	
	p, err := earley.GenerateParser(g)
	if err != nil {
		t.Error("parser generation failed: "+err.Error())
		return
	}
	fmt.Println(g.Name())
	
	lb := OpenLexerBuilder(g)
	lb.Token("ID").Expr(`([a-zA-Z][a-zA-Z0-9]*)`)
	lb.Token("EQ").Expr(`=`)
	lb.Token("PRINT").Expr(`print`)
	lb.Token("POW").Expr(`^`). 
	 	Value(func(part parser.GrammarParticle, match string) interface{} {
			return EXPONENTIATION
		})
	lb.Token("PLUS").Expr(`\+`). 
	 	Value(func(part parser.GrammarParticle, match string) interface{} {
			return ADDITION
		})
	lb.Token("MINUS").Expr(`-`). 
	 	Value(func(part parser.GrammarParticle, match string) interface{} {
			return SUBTRACTION
		})
	lb.Token("TIMES").Expr(`\*`). 
	 	Value(func(part parser.GrammarParticle, match string) interface{} {
			return MULTIPLICATION
		})
	lb.Token("DIV").Expr(`/`). 
	 	Value(func(part parser.GrammarParticle, match string) interface{} {
			return DIVISION
		})
	lb.Token("MOD").Expr(`%`). 
	 	Value(func(part parser.GrammarParticle, match string) interface{} {
			return MODULUS
		})
	lb.Token("LP").Expr(`\(`)
	lb.Token("RP").Expr(`\)`)
	lb.Token("NUM").Expr("(0|[1-9][0-9]*)").
		Value(func(part parser.GrammarParticle, match string) interface{} {
			n := big.NewInt(0)
			ten := big.NewInt(10)
			for i := len(match)-1; i >= 0; i-- {
				n.Mul(n,ten).Add(n, big.NewInt(int64(match[i]-'0')))
			}
			return n
		})
	lexer, err := lb.Build()
	if err != nil {
		t.Error("lexer build failed: "+err.Error())
		return
	}
	
	
	lexer.Reset(parser.NewStringReader(SAMPLE_PROGRAM))
	ast, err := p.Parse(lexer, nil)
	if err != nil {
		t.Error(err)
		return
	}
	ast = ast
}



