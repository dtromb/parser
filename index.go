package parser

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
)

type GrammarIndex interface {
	Name() string
	Grammar() Grammar
	Initialize(g Grammar) error
}

var GrammarIndexTypeTerm GrammarIndexType = reflect.TypeOf([]*termGrammarIndex{}).Elem()

type TermGrammarIndex interface {
	GrammarIndex
	HasTerm(name string) bool
	GetTerm(name string) (Term, error)
	GetTerminal(name string) (Term, error)
	GetNonterminal(name string) (Term, error)
	GetTerminalNames() []string
	GetNonterminalNames() []string
}

var GrammarIndexTypeProduction GrammarIndexType = reflect.TypeOf([]*productionNTGrammarIndex{}).Elem()

type ProductionGrammarIndex interface {
	GrammarIndex
	GetProductions(lhs Term) []ProductionRule
	GetInitialProduction() ProductionRule
	HasEpsilonProductions() bool
}

var GrammarIndexTypeNullability GrammarIndexType = reflect.TypeOf([]*nullabilityGrammarIndex{}).Elem()

type NullabilityGrammarIndex interface {
	GrammarIndex
	HasNullableNt() bool
	IsNullable(nt Term) bool
	GetNullableNonterminals() []Term
}

type GrammarIndexType reflect.Type

type IndexedGrammar interface {
	Grammar
	BaseGrammar() Grammar
	HasIndex(indexType GrammarIndexType) bool
	GetIndex(indexType GrammarIndexType) (GrammarIndex, error)
}

///

type stdIndexedGrammar struct {
	*stdGrammar
	indexCache map[string]GrammarIndex
}

func GetIndexedGrammar(g Grammar) IndexedGrammar {
	var base *stdGrammar
	if ig, ok := g.(IndexedGrammar); ok {
		return ig
	}
	if sg, ok := g.(*stdGrammar); ok {
		base = sg
	} else {
		base = copyGrammar(g)
	}
	return &stdIndexedGrammar{
		stdGrammar: base,
		indexCache: make(map[string]GrammarIndex),
	}
}

func (sig *stdIndexedGrammar) BaseGrammar() Grammar {
	return sig.stdGrammar
}

func (sig *stdIndexedGrammar) HasIndex(indexType GrammarIndexType) bool {
	indexTypeName := indexType.PkgPath() + "." + indexType.Name()
	if _, ok := sig.indexCache[indexTypeName]; ok {
		return true
	}
	return false
}

func (sig *stdIndexedGrammar) GetIndex(indexType GrammarIndexType) (GrammarIndex, error) {
	indexTypeName := indexType.Elem().PkgPath() + "." + indexType.Elem().Name()
	if idx, ok := sig.indexCache[indexTypeName]; ok {
		return idx, nil
	}
	var indexIf interface{}
	if indexType.Kind() == reflect.Ptr {
		indexIf = reflect.New(indexType.Elem()).Interface()
	} else {
		indexIf = reflect.New(indexType).Elem().Interface()
	}
	if reflect.TypeOf(indexIf).AssignableTo(reflect.TypeOf([]GrammarIndex{}).Elem()) {
		newIndex := indexIf.(GrammarIndex)
		err := newIndex.Initialize(sig)
		if err != nil {
			return nil, err
		}
		sig.indexCache[indexTypeName] = newIndex
		return newIndex, nil
	}
	return nil, errors.New("index type does not implement GrammarIndex")
}

type termGrammarIndex struct {
	g                  Grammar
	terminalsByName    map[string]Term
	nonterminalsByName map[string]Term
}

func (idx *termGrammarIndex) Name() string {
	return "term-index"
}

func (idx *termGrammarIndex) Grammar() Grammar {
	return idx.g
}

type strsort []string

func (ss strsort) Len() int           { return len(ss) }
func (ss strsort) Less(i, j int) bool { return ss[i] < ss[j] }
func (ss strsort) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }

func (idx *termGrammarIndex) Initialize(g Grammar) error {
	if idx.g != nil {
		return errors.New("already initialized")
	}
	idx.g = g
	idx.terminalsByName = make(map[string]Term)
	idx.nonterminalsByName = make(map[string]Term)
	for i := 0; i < g.NumTerminal(); i++ {
		t := g.Terminal(i)
		idx.terminalsByName[t.Name()] = t
		fmt.Println(" ----- terminal " + t.Name())
	}
	for i := 0; i < g.NumNonterminal(); i++ {
		nt := g.Nonterminal(i)
		idx.nonterminalsByName[nt.Name()] = nt
	}
	fmt.Println("term index init finished!")
	return nil
}

func (idx *termGrammarIndex) HasTerm(name string) bool {
	if _, has := idx.terminalsByName[name]; has {
		return true
	}
	if _, has := idx.nonterminalsByName[name]; has {
		return true
	}
	return false
}

func (idx *termGrammarIndex) GetTerm(name string) (Term, error) {
	if v, has := idx.terminalsByName[name]; has {
		return v, nil
	}
	if v, has := idx.nonterminalsByName[name]; has {
		return v, nil
	}
	return nil, errors.New(fmt.Sprintf("grammar term not found: '%s'", name))
}

func (idx *termGrammarIndex) GetTerminal(name string) (Term, error) {
	if v, has := idx.terminalsByName[name]; has {
		return v, nil
	}
	return nil, errors.New(fmt.Sprintf("grammar terminal not found: '%s'", name))
}

func (idx *termGrammarIndex) GetNonterminal(name string) (Term, error) {
	if v, has := idx.nonterminalsByName[name]; has {
		return v, nil
	}
	return nil, errors.New(fmt.Sprintf("grammar nonterminal not found: '%s'", name))
}

func (idx *termGrammarIndex) GetTerminalNames() []string {
	ret := strsort(make([]string, 0, idx.g.NumTerminal()))
	for k, _ := range idx.terminalsByName {
		ret = append(ret, k)
	}
	sort.Sort(ret)
	return ret
}

func (idx *termGrammarIndex) GetNonterminalNames() []string {
	ret := strsort(make([]string, 0, idx.g.NumNonterminal()))
	for k, _ := range idx.nonterminalsByName {
		ret = append(ret, k)
	}
	sort.Sort(ret)
	return ret
}

type productionNTGrammarIndex struct {
	g                Grammar
	productionsByLhs map[uint32][]ProductionRule
	initial          ProductionRule
	hasEpsilons      bool
}

func (pnt *productionNTGrammarIndex) Name() string {
	return "cfnt-production-index"
}

func (pnt *productionNTGrammarIndex) Grammar() Grammar {
	return pnt.g
}

type sortedprods []ProductionRule

func (sp sortedprods) Len() int      { return len(sp) }
func (sp sortedprods) Swap(i, j int) { sp[i], sp[j] = sp[j], sp[i] }
func (sp sortedprods) Less(i, j int) bool {
	a := sp[i]
	b := sp[j]
	if a.Lhs().Name() < b.Lhs().Name() {
		return true
	}
	if a.Lhs() == b.Lhs() {
		if a.RhsLen() < b.RhsLen() {
			return true
		}
		if a.RhsLen() == b.RhsLen() {
			for i := 0; i < a.RhsLen(); i++ {
				at := a.Rhs(i)
				bt := b.Rhs(i)
				if at.Terminal() && !bt.Terminal() {
					return true
				}
				if bt.Terminal() && !at.Terminal() {
					return false
				}
				if at.Name() < bt.Name() {
					return true
				}
				if bt.Name() < at.Name() {
					return false
				}
			}
		}
	}
	return false
}

func (pnt *productionNTGrammarIndex) Initialize(g Grammar) error {
	if pnt.g != nil {
		return errors.New("index already initialized")
	}
	pnt.g = g
	pnt.productionsByLhs = make(map[uint32][]ProductionRule)
	for i := 0; i < g.NumProductionRule(); i++ {
		pr := g.ProductionRule(i)
		if _, has := pnt.productionsByLhs[pr.Lhs().Id()]; !has {
			pnt.productionsByLhs[pr.Lhs().Id()] = []ProductionRule{pr}
		} else {
			pnt.productionsByLhs[pr.Lhs().Id()] = append(pnt.productionsByLhs[pr.Lhs().Id()], pr)
		}
		if pr.Lhs().Id() == pnt.g.Asterisk().Id() {
			if pnt.initial != nil {
				return errors.New("duplicate initial production in grammar")
			}
			if pr.RhsLen() != 2 || pr.Rhs(0).Terminal() || pr.Rhs(1).Id() != pnt.g.Bottom().Id() {
				return errors.New("incorrect initial production rule form")
			}
			pnt.initial = pr
		} else {
			if pr.Lhs().Terminal() || pr.Lhs().Special() {
				return errors.New("invalid LHS in grammar rule")
			}
			var sawEps bool
			for i := 0; i < pr.RhsLen(); i++ {
				rht := pr.Rhs(i)
				if rht.Special() {
					if rht.Id() != pnt.g.Epsilon().Id() {
						return errors.New("invalid term in RHS of grammar rule")
					}
					if sawEps {
						return errors.New("consecutive epsilons in RHS of grammar rule")
					}
					sawEps = true
					pnt.hasEpsilons = true
				} else {
					sawEps = false
				}
			}
		}
	}
	for k, ps := range pnt.productionsByLhs {
		sp := sortedprods(make([]ProductionRule, len(ps)))
		copy(sp, ps)
		sort.Sort(sp)
		pnt.productionsByLhs[k] = sp
		fmt.Printf("index: %d productions for ID %d\n", len(sp), k)
	}
	if pnt.initial == nil {
		return errors.New("no initial production in grammar")
	}
	return nil
}

func (pnt *productionNTGrammarIndex) GetProductions(lhs Term) []ProductionRule {
	if v, has := pnt.productionsByLhs[lhs.Id()]; has {
		ret := make([]ProductionRule, len(v))
		copy(ret, v)
		return ret
	} else {
		return []ProductionRule{}
	}
}

func (pnt *productionNTGrammarIndex) GetInitialProduction() ProductionRule {
	return pnt.initial
}

func (pnt *productionNTGrammarIndex) HasEpsilonProductions() bool {
	return pnt.hasEpsilons
}

type nullabilityGrammarIndex struct {
	g          Grammar
	nullableNt map[uint32]Term
}

func (ni *nullabilityGrammarIndex) Name() string {
	return "nullability-index"
}

func (ni *nullabilityGrammarIndex) Grammar() Grammar {
	return ni.g
}

func (ni *nullabilityGrammarIndex) Initialize(g Grammar) error {
	if ni.g != nil {
		return errors.New("index already initialized")
	}
	ni.g = g
	ni.nullableNt = make(map[uint32]Term)
	changed := true
	for changed {
		changed = false
		for i := 0; i < g.NumProductionRule(); i++ {
			pr := g.ProductionRule(i)
			if _, has := ni.nullableNt[pr.Lhs().Id()]; has {
				continue
			}
			nullDeriv := true
			for j := 0; j < pr.RhsLen(); j++ {
				prt := pr.Rhs(j)
				if prt.Id() == g.Epsilon().Id() {
					continue
				}
				if prt.Terminal() {
					nullDeriv = false
					break
				} else {
					if _, has := ni.nullableNt[prt.Id()]; !has {
						nullDeriv = false
						break
					}
				}
			}
			if nullDeriv {
				if _, has := ni.nullableNt[pr.Lhs().Id()]; !has {
					changed = true
				}
				ni.nullableNt[pr.Lhs().Id()] = pr.Lhs()
			}
		}
	}
	return nil
}

func (ni *nullabilityGrammarIndex) HasNullableNt() bool {
	return len(ni.nullableNt) > 0
}

func (ni *nullabilityGrammarIndex) IsNullable(nt Term) bool {
	if _, has := ni.nullableNt[nt.Id()]; has {
		return true
	}
	return nt.Id() == ni.g.Epsilon().Id()
}

func (ni *nullabilityGrammarIndex) GetNullableNonterminals() []Term {
	ret := make([]Term, 0, len(ni.nullableNt))
	for _, nt := range ni.nullableNt {
		ret = append(ret, nt)
	}
	return ret
}
