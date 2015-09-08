package index

import (
	"reflect"
	"github.com/dtromb/parser"
)

// Grammar class index.  Basic grammar type detection.

type GrammarClassIndex struct {
	g parser.Grammar 
	gclass parser.ProductionClass
	greg parser.Regularity
}

const(
	GRAMMAR_CLASS_INDEX	string = "grammar-class"
)

func init() {
	parser.RegisterGrammarIndexType(reflect.TypeOf([]GrammarClassIndex{}).Elem())
}

func (gci *GrammarClassIndex) IndexName() string {
	return GRAMMAR_CLASS_INDEX
}
	
func (gci *GrammarClassIndex) Class() parser.ProductionClass {
	return gci.gclass
}

func (gci *GrammarClassIndex) Regularity() parser.Regularity {
	return gci.greg
}

func (gci *GrammarClassIndex) Grammar() parser.Grammar {
	return gci.g
}

func (gci *GrammarClassIndex) Initialize(g parser.Grammar) error {
	gci.g = g
	gci.gclass = parser.CONSTANT
	gci.greg = parser.STRICT_UNITARY
	for _, p := range g.Productions() {
		if p.Lhs(0).Asterisk() {
			continue
		}
		preg := parser.GetProductionClass(p)
		if preg > gci.gclass {
			gci.gclass = preg
			if gci.gclass > parser.REGULAR {
				gci.greg = parser.NONREGULAR
			}
		}
		if gci.gclass <= parser.REGULAR {
			gci.greg = gci.greg.Join(parser.GetProductionRegularity(p))
		}
	}
	return nil
}

