package parser

import (
	"errors"
	"reflect"
)

type GrammarIndex interface {
	IndexName() string
	Grammar() Grammar
	Initialize(g Grammar) error
}

type IndexedGrammar struct {
	baseGrammar Grammar
	indexes map[string]GrammarIndex
}

var global_GrammarIndexTypes map[string]reflect.Type = make(map[string]reflect.Type)

func RegisterGrammarIndexType(typ reflect.Type) error {
	var indexName string
	proto := reflect.New(typ)
	if idx, ok := proto.Interface().(GrammarIndex); !ok {
		return errors.New(typ.Name()+" is not a GrammarIndex")
	} else {
		indexName = idx.IndexName()
	}
	if _, has := global_GrammarIndexTypes[indexName]; has {
		return errors.New("grammar index '"+indexName+"' already registered")
	}
	global_GrammarIndexTypes[indexName] = typ
	return nil
}

func GetIndexedGrammar(grammar Grammar) *IndexedGrammar {
	if g, ok := grammar.(*IndexedGrammar); ok {
		return g
	}
	return &IndexedGrammar {
		baseGrammar: grammar,
		indexes: make(map[string]GrammarIndex),
	}
}

func (ig *IndexedGrammar) GetIndex(name string) (GrammarIndex, error) {
	index, has := ig.indexes[name]
	if has {
		return index, nil
	}
	indexType, has := global_GrammarIndexTypes[name]
	if !has {
		return nil, errors.New("unknown grammar index type '"+name+"'")
	}
	index = reflect.New(indexType).Interface().(GrammarIndex)
	if err := index.Initialize(ig); err != nil {
		return nil, err
	}
	return index, nil
}

func (g *IndexedGrammar) Name() string { return g.baseGrammar.Name() }
func (g *IndexedGrammar) NumNonterminals() int { return g.baseGrammar.NumNonterminals() }
func (g *IndexedGrammar) Nonterminal(idx int) GrammarParticle { return g.baseGrammar.Nonterminal(idx) }
func (g *IndexedGrammar) Nonterminals() []GrammarParticle { return g.baseGrammar.Nonterminals() }
func (g *IndexedGrammar) NumTerminals() int { return g.baseGrammar.NumTerminals() }
func (g *IndexedGrammar) Terminal(idx int) GrammarParticle { return g.baseGrammar.Terminal(idx) }
func (g *IndexedGrammar) Terminals() []GrammarParticle { return g.baseGrammar.Terminals() }
func (g *IndexedGrammar) Epsilon() GrammarParticle { return g.baseGrammar.Epsilon() }
func (g *IndexedGrammar) Asterisk() GrammarParticle { return g.baseGrammar.Asterisk() }
func (g *IndexedGrammar) Bottom() GrammarParticle { return g.baseGrammar.Bottom() }
func (g *IndexedGrammar) NumProductions() int { return g.baseGrammar.NumProductions() }
func (g *IndexedGrammar) Production(idx int) Production { return g.baseGrammar.Production(idx) }
func (g *IndexedGrammar) Productions() []Production { return g.baseGrammar.Productions() }