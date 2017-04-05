package parser

import (
	"errors"
)

type Term interface {
	Hashable
	Grammar() Grammar
	Name() string
	Id() uint32
	Terminal() bool
	Special() bool
}

type ProductionRule interface {
	Hashable
	Grammar() Grammar
	Id() uint32
	Lhs() Term
	RhsLen() int
	Rhs(idx int) Term
	RhsSlice() []Term
}

type Grammar interface {
	NumTerminal() int
	Terminal(idx int) Term
	NumNonterminal() int
	Nonterminal(idx int) Term
	Asterisk() Term
	Epsilon() Term
	Bottom() Term
	NumProductionRule() int
	ProductionRule(idx int) ProductionRule
}

type GrammarBuilder interface {
	Terminal(t string) GrammarBuilder
	Nonterminal(t string) GrammarBuilder
	Rule(lhsNt string) GrammarBuilder
	Build() (Grammar, error)
}

///

type stdGrammar struct {
	terminals    []*stdTerm
	nonterminals []*stdTerm
	productions  []*stdProduction
	asterisk     *stdTerm
	epsilon      *stdTerm
	bottom       *stdTerm
}

type stdTerm struct {
	grammar *stdGrammar
	nonterm bool
	special bool
	name    string
	id      uint32
}

type stdProduction struct {
	grammar *stdGrammar
	id      uint32
	lhs     Term
	rhs     []Term
	hc      uint32
}

func (sg *stdGrammar) NumTerminal() int {
	return len(sg.terminals)
}

func (sg *stdGrammar) Terminal(idx int) Term {
	if idx < 0 || idx >= len(sg.terminals) {
		panic("terminal index out of range")
	}
	return sg.terminals[idx]
}

func (sg *stdGrammar) NumNonterminal() int {
	return len(sg.nonterminals)
}

func (sg *stdGrammar) Nonterminal(idx int) Term {
	if idx < 0 || idx >= len(sg.terminals) {
		panic("nonterminal index out of range")
	}
	return sg.nonterminals[idx]
}

func (sg *stdGrammar) Asterisk() Term {
	return sg.asterisk
}

func (sg *stdGrammar) Epsilon() Term {
	return sg.epsilon
}

func (sg *stdGrammar) Bottom() Term {
	return sg.bottom
}

func (sg *stdGrammar) NumProductionRule() int {
	return len(sg.productions)
}

func (sg *stdGrammar) ProductionRule(idx int) ProductionRule {
	if idx < 0 || idx >= len(sg.productions) {
		panic("production rule index out of range")
	}
	return sg.productions[idx]
}

func (st *stdTerm) Grammar() Grammar {
	return st.grammar
}

func (st *stdTerm) HashCode() uint32 {
	return st.id
}

func (st *stdTerm) Equals(o interface{}) bool {
	if k, ok := o.(Term); ok {
		return k.Id() == st.id && k.Grammar() == st.grammar
	}
	return false
}

func (st *stdTerm) Name() string {
	return st.name
}

func (st *stdTerm) Id() uint32 {
	return st.id
}

func (st *stdTerm) Terminal() bool {
	return !st.nonterm && !st.special
}

func (st *stdTerm) Special() bool {
	return st.special
}

func (sp *stdProduction) HashCode() uint32 {
	if sp.hc == 0 {
		sp.hc = 0x10000000 ^ sp.lhs.HashCode()
		for _, t := range sp.rhs {
			sp.hc = (sp.hc >> 7) | (sp.hc << 25)
			sp.hc ^= t.HashCode()
		}
	}
	return sp.hc
}

func (sp *stdProduction) Equals(o interface{}) bool {
	if p, ok := o.(ProductionRule); ok {
		if p.HashCode() != sp.HashCode() {
			return false
		}
		if ssp, ok := p.(*stdProduction); ok && ssp == sp {
			return true
		}
		if p.Grammar() != sp.grammar {
			return false
		}
		if !p.Lhs().Equals(sp.lhs) {
			return false
		}
		if p.RhsLen() != len(sp.rhs) {
			return false
		}
		for i, r := range sp.rhs {
			if !r.Equals(p.Rhs(i)) {
				return false
			}
		}
		return true
	}
	return false
}

func (sp *stdProduction) Grammar() Grammar {
	return sp.grammar
}

func (sp *stdProduction) Id() uint32 {
	return sp.id
}

func (sp *stdProduction) Lhs() Term {
	return sp.lhs
}

func (sp *stdProduction) RhsLen() int {
	return len(sp.rhs)
}

func (sp *stdProduction) Rhs(idx int) Term {
	if idx < 0 || idx >= len(sp.rhs) {
		panic("production rule RHS index out of range")
	}
	return sp.rhs[idx]
}

func (sp *stdProduction) RhsSlice() []Term {
	ret := make([]Term, len(sp.rhs))
	copy(ret, sp.rhs)
	return ret
}

func copyGrammar(g Grammar) *stdGrammar {
	sg := &stdGrammar{}
	sg.terminals = make([]*stdTerm, g.NumTerminal())
	sg.nonterminals = make([]*stdTerm, g.NumNonterminal())
	sg.productions = make([]*stdProduction, g.NumProductionRule())
	sg.asterisk = &stdTerm{
		grammar: sg,
		nonterm: true,
		special: true,
		name:    "`*",
		id:      1,
	}
	sg.epsilon = &stdTerm{
		grammar: sg,
		nonterm: false,
		special: true,
		name:    "`e",
		id:      2,
	}
	sg.asterisk = &stdTerm{
		grammar: sg,
		nonterm: false,
		special: true,
		name:    "`.",
		id:      3,
	}
	idmap := make(map[uint32]Term)
	nextId := uint32(100)
	for i := 0; i < len(sg.terminals); i++ {
		t := g.Terminal(i)
		sg.terminals[i] = &stdTerm{
			grammar: sg,
			nonterm: false,
			special: false,
			name:    t.Name(),
			id:      nextId,
		}
		nextId++
		if _, has := idmap[t.Id()]; has {
			panic("duplicate term id in grammar")
		}
		idmap[t.Id()] = sg.terminals[i]
	}
	for i := 0; i < len(sg.nonterminals); i++ {
		t := g.Nonterminal(i)
		sg.nonterminals[i] = &stdTerm{
			grammar: sg,
			nonterm: true,
			special: false,
			name:    t.Name(),
			id:      nextId,
		}
		nextId++
		if _, has := idmap[t.Id()]; has {
			panic("duplicate term id in grammar")
		}
		idmap[t.Id()] = sg.nonterminals[i]
	}
	for i := 0; i < len(sg.productions); i++ {
		p := g.ProductionRule(i)
		sg.productions[i] = &stdProduction{
			grammar: sg,
			id:      nextId,
		}
		nextId++
		var ok bool
		if sg.productions[i].lhs, ok = idmap[p.Lhs().Id()]; !ok {
			panic("production references unknown term id")
		}
		sg.productions[i].rhs = make([]Term, p.RhsLen())
		for j := 0; j < p.RhsLen(); j++ {
			if sg.productions[i].rhs[j], ok = idmap[p.Rhs(j).Id()]; !ok {
				panic("production references unknown term id")
			}
		}
	}
	return sg
}

func TermToString(t Term) string {
	switch t.Id() {
	case t.Grammar().Asterisk().Id():
		{
			return "`*"
		}
	case t.Grammar().Bottom().Id():
		{
			return "`."
		}
	case t.Grammar().Epsilon().Id():
		{
			return "`e"
		}
	}
	if t.Terminal() {
		return t.Name()
	}
	return "<" + t.Name() + ">"
}

func ProductionRuleToString(pr ProductionRule) string {
	var buf []byte
	buf = []byte(TermToString(pr.Lhs()))
	buf = append(buf, " := "...)
	for i := 0; i < pr.RhsLen(); i++ {
		term := pr.Rhs(i)
		//fmt.Printf("%d: %s\n", i, TermToString(term))
		buf = append(buf, TermToString(term)...)
		buf = append(buf, byte(' '))
	}
	buf = buf[0 : len(buf)-1]
	return string(buf)
}

type stdGrammarBuilder struct {
	terminals     map[string]*prototypeTerm
	nonterminals  map[string]*prototypeTerm
	finishedRules map[uint32][]*prototypeProduction
	openRule      *prototypeProduction
	initialRule   *prototypeProduction
	nextId        uint32
	grammar       *prototypeGrammar
	built         bool
	builtGrammar  Grammar
}

type prototypeGrammar struct {
	*stdGrammarBuilder
}

type prototypeTerm struct {
	*stdTerm
	builder *stdGrammarBuilder
}

type prototypeProduction struct {
	*stdProduction
	builder *stdGrammarBuilder
}

func NewGrammarBuilder() GrammarBuilder {
	gb := &stdGrammarBuilder{
		nextId:        100,
		terminals:     make(map[string]*prototypeTerm),
		nonterminals:  make(map[string]*prototypeTerm),
		finishedRules: make(map[uint32][]*prototypeProduction),
		grammar:       &prototypeGrammar{},
	}
	gb.grammar.stdGrammarBuilder = gb
	gb.nonterminals["`*"] = &prototypeTerm{
		builder: gb,
		stdTerm: &stdTerm{
			nonterm: true,
			special: true,
			name:    "`*",
			id:      1,
		},
	}
	gb.terminals["`e"] = &prototypeTerm{
		builder: gb,
		stdTerm: &stdTerm{
			nonterm: false,
			special: true,
			name:    "`e",
			id:      2,
		},
	}
	gb.terminals["`."] = &prototypeTerm{
		builder: gb,
		stdTerm: &stdTerm{
			nonterm: false,
			special: true,
			name:    "`.",
			id:      3,
		},
	}
	return gb
}

func (sg *stdGrammarBuilder) isValidSymbolCharacter(c byte) bool {
	return ((c >= 'a') && (c <= 'z')) ||
		((c >= 'A') && (c <= 'Z')) ||
		((c >= '0') && (c <= '9')) ||
		(c == '-') || (c == '_')
}

func (sg *stdGrammarBuilder) getTerm(name string, terminal bool) (Term, error) {
	if terminal {
		if term, has := sg.terminals[name]; has {
			return term, nil
		} else {
			for _, c := range []byte(name) {
				if !sg.isValidSymbolCharacter(c) {
					return nil, errors.New("name argument is an invalid term name")
				}
			}
		}
		sg.terminals[name] = &prototypeTerm{
			builder: sg,
			stdTerm: &stdTerm{
				nonterm: false,
				special: false,
				name:    name,
				id:      sg.nextId,
			},
		}
		sg.nextId++
		return sg.terminals[name], nil
	} else {
		if term, has := sg.nonterminals[name]; has {
			return term, nil
		} else {
			for _, c := range []byte(name) {
				if !sg.isValidSymbolCharacter(c) {
					return nil, errors.New("name argument is an invalid term name")
				}
			}
		}
		sg.nonterminals[name] = &prototypeTerm{
			builder: sg,
			stdTerm: &stdTerm{
				nonterm: true,
				special: false,
				name:    name,
				id:      sg.nextId,
			},
		}
		sg.nextId++
		return sg.nonterminals[name], nil
	}
}

func (sg *stdGrammarBuilder) Terminal(t string) GrammarBuilder {
	if sg.openRule == nil {
		panic("Terminal() called before Rule()")
	}
	addTerm, err := sg.getTerm(t, true)
	if err != nil {
		panic("Terminal() " + err.Error())
	}
	sg.openRule.rhs = append(sg.openRule.rhs, addTerm)
	return sg
}

func (sg *stdGrammarBuilder) Nonterminal(nt string) GrammarBuilder {
	if sg.openRule == nil {
		panic("Nonterminal() called before Rule()")
	}
	addTerm, err := sg.getTerm(nt, false)
	if err != nil {
		panic("Nonterminal() " + err.Error())
	}
	sg.openRule.rhs = append(sg.openRule.rhs, addTerm)
	return sg
}

func (sg *stdGrammarBuilder) Rule(lhsNt string) GrammarBuilder {
	if sg.openRule != nil {
		if sg.openRule.lhs.Name() == "`*" {
			if sg.openRule.RhsLen() != 2 ||
				sg.openRule.Rhs(0).Terminal() ||
				sg.openRule.Rhs(1).Name() != "`." {
				panic("invalid inital rule format: " + ProductionRuleToString(sg.openRule))
			}
			if sg.initialRule != nil && !sg.built {
				panic("duplicate initial rule")
			}
			sg.initialRule = sg.openRule
		}
		hc := sg.openRule.HashCode()
		if m, has := sg.finishedRules[hc]; !has {
			sg.finishedRules[hc] = []*prototypeProduction{sg.openRule}
		} else {
			for _, pr := range m {
				if pr.Equals(sg.openRule) {
					panic("duplicate rule")
				}
			}
			sg.finishedRules[hc] = append(sg.finishedRules[hc], sg.openRule)
		}
	}
	lhsTerm, err := sg.getTerm(lhsNt, false)
	if err != nil {
		panic("Rule() " + err.Error())
	}
	sg.openRule = &prototypeProduction{
		builder: sg,
		stdProduction: &stdProduction{
			id:  sg.nextId,
			rhs: []Term{},
			lhs: lhsTerm,
		},
	}
	sg.nextId++
	return sg
}

func (sg *stdGrammarBuilder) Build() (Grammar, error) {
	if sg.built {
		return sg.builtGrammar, nil
	}
	sg.built = true
	sg.Rule("`*")
	grammar := &stdGrammar{
		terminals:    make([]*stdTerm, 0, len(sg.terminals)-1),
		nonterminals: make([]*stdTerm, 0, len(sg.nonterminals)-2),
		productions:  make([]*stdProduction, 0, sg.grammar.NumProductionRule()),
		asterisk:     sg.nonterminals["`*"].stdTerm,
		epsilon:      sg.terminals["`e"].stdTerm,
		bottom:       sg.terminals["`."].stdTerm,
	}
	grammar.asterisk.grammar = grammar
	grammar.epsilon.grammar = grammar
	grammar.bottom.grammar = grammar
	for _, t := range sg.terminals {
		t.stdTerm.grammar = grammar
		grammar.terminals = append(grammar.terminals, t.stdTerm)
	}
	for _, nt := range sg.nonterminals {
		nt.stdTerm.grammar = grammar
		grammar.nonterminals = append(grammar.nonterminals, nt.stdTerm)
	}
	for _, m := range sg.finishedRules {
		for _, pr := range m {
			grammar.productions = append(grammar.productions, pr.stdProduction)
			pr.grammar = grammar
		}
	}
	sg.builtGrammar = grammar
	return grammar, nil
}

func (pg *prototypeGrammar) NumTerminal() int {
	return len(pg.nonterminals)
}

func (pg *prototypeGrammar) Terminal(idx int) Term {
	i := 0
	for _, term := range pg.terminals {
		if i < idx {
			i++
		} else {
			return term
		}
	}
	return nil
}

func (pg *prototypeGrammar) NumNonterminal() int {
	return len(pg.terminals)
}

func (pg *prototypeGrammar) Nonterminal(idx int) Term {
	i := 0
	for _, term := range pg.nonterminals {
		if i < idx {
			i++
		} else {
			return term
		}
	}
	return nil
}

func (pg *prototypeGrammar) Asterisk() Term {
	return pg.nonterminals["`*"]
}

func (pg *prototypeGrammar) Epsilon() Term {
	return pg.terminals["`e"]
}

func (pg *prototypeGrammar) Bottom() Term {
	return pg.terminals["`."]
}

func (pg *prototypeGrammar) NumProductionRule() int {
	c := 0
	for _, m := range pg.finishedRules {
		c += len(m)
	}
	return c
}

func (pg *prototypeGrammar) ProductionRule(idx int) ProductionRule {
	c := 0
	for _, m := range pg.finishedRules {
		for i, pr := range m {
			if c+i == idx {
				return pr
			}
		}
		c += len(m)
	}
	return nil
}

func (pt *prototypeTerm) Grammar() Grammar {
	return pt.builder.grammar
}

func (pt *prototypeProduction) Grammar() Grammar {
	return pt.builder.grammar
}
