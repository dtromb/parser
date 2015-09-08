package parser

import (
	"errors"
	c "github.com/dtromb/collections"
	"github.com/dtromb/collections/tree"
)

type ParticleType int
const (
	TERMINAL ParticleType = iota
	NONTERMINAL
	ASTERISK
	EPSILON	
	BOTTOM
)

type ProductionClass int
const (
	CONSTANT		ProductionClass = iota
	REGULAR
	CONTEXT_FREE
	CONTEXT_SENSITIVE
	GENERAL
)
func (pc ProductionClass) String() string {
	switch(pc) {
		case CONSTANT: return "constant"
		case REGULAR: return "regular"
		case CONTEXT_FREE: return "context-free"
		case CONTEXT_SENSITIVE: return "context-sensitive"
		case GENERAL: return "general"
	}
	panic("invalid grammar class")
}

// Why we think of these comparisons this way, I have no idea...
func (pc ProductionClass) ContextSensitive() bool {
	return pc >= CONTEXT_SENSITIVE
}

func (pc ProductionClass) ContextFree() bool {
	return pc <= CONTEXT_FREE
}

func (pc ProductionClass) Regular() bool {
	return pc <= REGULAR
}


type Regularity int 
const (
	LEFT			Regularity = iota
	STRICT_LEFT
	RIGHT
	STRICT_RIGHT
	UNITARY
	STRICT_UNITARY
	NONREGULAR
)
func (r Regularity) String() string {
	switch (r) {
		case LEFT: return "left-regular"
		case STRICT_LEFT: return "strict-left-regular"
		case RIGHT: return "right-regular"
		case STRICT_RIGHT: return "strict-right-regular"
		case UNITARY: return "symmetric-regular"
		case STRICT_UNITARY: return "strict-symmetric-regular"
		case NONREGULAR: return "nonregular"
	}
	panic("invalid regularity")
}

func (r Regularity) Left() bool {
	return r == LEFT || r == STRICT_LEFT || r == UNITARY || r == STRICT_UNITARY
}

func (r Regularity) Right() bool {
	return r == RIGHT || r == STRICT_RIGHT || r == UNITARY || r == STRICT_UNITARY
}

func (r Regularity) Strict() bool {
	return r == STRICT_LEFT || r == STRICT_RIGHT || r ==  STRICT_UNITARY
}

func (r Regularity) Unitary() bool {
	return r == UNITARY || r == STRICT_UNITARY
}

func (r Regularity) Regular() bool {
	return r != NONREGULAR
}

func (r1 Regularity) Join(r2 Regularity) Regularity {
	if r1.Unitary() && r2.Unitary() {
		if r1.Strict() && r2.Strict() {
			return STRICT_UNITARY
		}
		return UNITARY
	}
	if r1.Left() && r2.Left() {
		if r1.Strict() && r2.Strict() {
			return STRICT_LEFT
		}
		return LEFT
	}
	if r1.Right() && r2.Right() {
		if r1.Strict() && r2.Strict() {
			return STRICT_RIGHT
		}
		return RIGHT
	}
	return NONREGULAR
}

type Grammar interface {
	Name() string
	NumNonterminals() int
	Nonterminal(idx int) GrammarParticle
	Nonterminals() []GrammarParticle
	NumTerminals() int
	Terminal(idx int) GrammarParticle
	Terminals() []GrammarParticle
	Epsilon() GrammarParticle
	Asterisk() GrammarParticle
	Bottom() GrammarParticle
	NumProductions() int
	Production(idx int) Production
	Productions() []Production
}

type GrammarParticle interface {
	c.Comparable
	Name() string
	Terminal() bool
	Nonterminal() bool
	Epsilon() bool
	Asterisk() bool
	Bottom() bool
	Grammar() Grammar
	Type() ParticleType
	String() string
}

type Production interface {
	c.Comparable
	LhsCopy() []GrammarParticle
	RhsCopy() []GrammarParticle
	LhsLen() int
	RhsLen() int
	Lhs(idx int) GrammarParticle
	Rhs(idx int) GrammarParticle
	Substitute(map[GrammarParticle]GrammarParticle) (Production,bool)
	String() string
}

type GrammarBuilder struct {
	name string
	nonterms map[string]GrammarParticle
	terms map[string]GrammarParticle
	epsilon GrammarParticle
	asterisk GrammarParticle
	bottom GrammarParticle
	rules tree.Tree
	ruleOpen bool
	lhs []GrammarParticle
	initialRule Production
	usedEpsilon bool
}


type BasicGrammar struct {
	name string
	nonterminals []GrammarParticle
	terminals []GrammarParticle
	epsilon GrammarParticle
	asterisk GrammarParticle
	productions []Production	
	bottom GrammarParticle
}

func CompareGrammarParticles(a GrammarParticle, b GrammarParticle) int8 {
	aType := a.Type()
	bType := b.Type()
	if aType > bType {
		return 1
	}
	if aType < bType {
		return -1
	}
	switch(aType) {
		case NONTERMINAL: fallthrough
		case TERMINAL: {
			aName := a.Name()
			bName := b.Name()
			if aName > bName {
				return 1
			}
			if aName < bName {
				return -1
			}
			return 0
		}
	}
	return 0
}

type BasicProduction struct {
	grammar Grammar
	lhs []GrammarParticle
	rhs []GrammarParticle
}

func (bp *BasicProduction) LhsCopy() []GrammarParticle {
	lhs := make([]GrammarParticle, 0, len(bp.lhs))
	copy(lhs, bp.lhs)
	return lhs
}

func (bp *BasicProduction) RhsCopy() []GrammarParticle {
	rhs := make([]GrammarParticle, 0, len(bp.rhs))
	copy(rhs, bp.rhs)
	return rhs
}

func (bp *BasicProduction) Lhs(idx int) GrammarParticle {
	return bp.lhs[idx]
}

func (bp *BasicProduction) Rhs(idx int) GrammarParticle {
	return bp.rhs[idx]
}

func (bp *BasicProduction) LhsLen() int {
	return len(bp.lhs)
}

func (bp *BasicProduction) RhsLen() int {
	return len(bp.rhs)
}

func (bp *BasicProduction) String() string {
	var buf []byte
	for _, k := range bp.lhs {
		buf = append(buf, k.String()...)
		buf = append(buf, " "...)
	}
	buf = append(buf, "-> "...)
	for i, k := range bp.rhs {
		buf = append(buf, k.String()...)
		if i < len(bp.rhs)-1 {
			buf = append(buf, " "...)
		}
	}
	return string(buf)
}

func (bp *BasicProduction) Substitute(smap map[GrammarParticle]GrammarParticle) (Production, bool) {
	var changed bool
	bpc := &BasicProduction{
		grammar:bp.grammar, 
		lhs:make([]GrammarParticle, len(bp.lhs)), 
		rhs:make([]GrammarParticle, len(bp.rhs)),
	}
	for i, p := range bp.lhs {
		if rp, has := smap[p]; has {
			bpc.lhs[i] = rp
			changed = true
		} else {
			bpc.lhs[i] = p
		}
	}
	for i, p := range bp.rhs {
		if rp, has := smap[p]; has {
			bpc.rhs[i] = rp
			changed = true
		} else {
			bpc.rhs[i] = p
		}
	}
	return bpc, changed
}

func (bp *BasicProduction) CompareTo(o c.Comparable) int8 {
	op := o.(Production)
	if op == Production(bp) {
		return 0
	}
	
	llen := len(bp.lhs)
	opllen := op.LhsLen()
	
	if llen < opllen {
		return -1
	}
	if llen > opllen {
		return 1
	}
	for i, p := range bp.lhs {
		r := p.CompareTo(op.Lhs(i))
		if r != 0 {
			return r
		}
	} 
	
	rlen := len(bp.rhs)
	oprlen := op.RhsLen()
	
	if rlen < oprlen {
		return -1
	}
	if rlen > oprlen {
		return 1
	}
	for i, p := range bp.rhs {
		r := p.CompareTo(op.Rhs(i))
		if r != 0 {
			return r
		}
	} 
	
	return 0
}


type GenericBottom struct {
	grammar Grammar
}

func (gb *GenericBottom) Name() string { return "`." }
func (gb *GenericBottom) Terminal() bool { return false }
func (gb *GenericBottom) Nonterminal() bool { return false }
func (gb *GenericBottom) Epsilon() bool { return false }
func (gb *GenericBottom) Asterisk() bool { return false }
func (gb *GenericBottom) Bottom() bool { return true }
func (gb *GenericBottom) Grammar() Grammar { return gb.grammar }
func (gb *GenericBottom) Type() ParticleType { return TERMINAL }
func (gb *GenericBottom) String() string { return gb.Name() }
func (gb *GenericBottom) CompareTo(o c.Comparable) int8 { 
	return CompareGrammarParticles(gb,o.(GrammarParticle))
}

type GrammarTerminal interface {
	GrammarParticle
	Value(val interface{}) *ValueTerminal
}

type GenericTerminal struct {
	GrammarTerminal
	grammar Grammar
	name string
}

func (gt *GenericTerminal) Value(val interface{}) *ValueTerminal {
	return &ValueTerminal{generic:gt, value: val}
}
func (gt *GenericTerminal) Name() string { return gt.name }
func (gt *GenericTerminal) Terminal() bool { return true }
func (gt *GenericTerminal) Nonterminal() bool { return false }
func (gt *GenericTerminal) Epsilon() bool { return false }
func (gt *GenericTerminal) Asterisk() bool { return false }
func (gt *GenericTerminal) Bottom() bool { return false }
func (gt *GenericTerminal) Grammar() Grammar { return gt.grammar }
func (gt *GenericTerminal) Type() ParticleType { return TERMINAL }
func (gt *GenericTerminal) String() string { return gt.name }
func (gt *GenericTerminal) CompareTo(o c.Comparable) int8 { 
	return CompareGrammarParticles(gt,o.(GrammarParticle))
}

func NewValueTerminal(generic GrammarParticle, value interface{}) *ValueTerminal {
	return &ValueTerminal{
		generic: generic,
		value: value,
	}
}

type ValueTerminal struct {
	generic GrammarParticle
	value interface{}
}
func (vt *ValueTerminal) Value() interface{} { return vt.value }
func (vt *ValueTerminal) CanonicalTerminal() GrammarParticle { return vt.generic }
func (vt *ValueTerminal) Name() string { return vt.generic.Name() }
func (vt *ValueTerminal) Terminal() bool { return true }
func (vt *ValueTerminal) Nonterminal() bool { return false }
func (vt *ValueTerminal) Epsilon() bool { return false }
func (vt *ValueTerminal) Asterisk() bool { return false }
func (vt *ValueTerminal) Bottom() bool { return false }
func (vt *ValueTerminal) Grammar() Grammar { return vt.generic.Grammar() }
func (vt *ValueTerminal) Type() ParticleType { return TERMINAL }
func (vt *ValueTerminal) String() string { return vt.Name() }
func (vt *ValueTerminal) CompareTo(o c.Comparable) int8 { 
	return CompareGrammarParticles(vt.generic,o.(GrammarParticle))
}

type GenericNonterminal struct {
	grammar Grammar
	name string
}
func (gt *GenericNonterminal) Name() string { return gt.name }
func (gt *GenericNonterminal) Terminal() bool { return false }
func (gt *GenericNonterminal) Nonterminal() bool { return true }
func (gt *GenericNonterminal) Epsilon() bool { return false }
func (gt *GenericNonterminal) Asterisk() bool { return false }
func (gt *GenericNonterminal) Bottom() bool { return false }
func (gt *GenericNonterminal) Grammar() Grammar { return gt.grammar }
func (gt *GenericNonterminal) Type() ParticleType { return NONTERMINAL }
func (gt *GenericNonterminal) String() string { return "<"+gt.name+">" }
func (gt *GenericNonterminal) CompareTo(o c.Comparable) int8 { 
	return CompareGrammarParticles(gt,o.(GrammarParticle))
}

type GenericEpsilon struct {
	grammar Grammar
}
func (ge *GenericEpsilon) Name() string { return "`e"}
func (ge *GenericEpsilon) Terminal() bool { return false }
func (ge *GenericEpsilon) Nonterminal() bool { return false }
func (ge *GenericEpsilon) Epsilon() bool { return true }
func (ge *GenericEpsilon) Asterisk() bool { return false }
func (ge *GenericEpsilon) Bottom() bool { return false }
func (ge *GenericEpsilon) Grammar() Grammar { return ge.grammar }
func (ge *GenericEpsilon) Type() ParticleType { return EPSILON }
func (ge *GenericEpsilon) String() string { return ge.Name() }
func (ge *GenericEpsilon) CompareTo(o c.Comparable) int8 { 
	return CompareGrammarParticles(ge,o.(GrammarParticle))
}
	
type GenericAsterisk struct {
	grammar Grammar
}
func (ga *GenericAsterisk) Name() string { return "`*"}
func (ga *GenericAsterisk) Terminal() bool { return false }
func (ga *GenericAsterisk) Nonterminal() bool { return false }
func (ga *GenericAsterisk) Epsilon() bool { return false }
func (ga *GenericAsterisk) Asterisk() bool { return true }
func (ga *GenericAsterisk) Bottom() bool { return false }
func (ga *GenericAsterisk) Grammar() Grammar { return ga.grammar }
func (ga *GenericAsterisk) String() string { return ga.Name() }
func (ga *GenericAsterisk) Type() ParticleType { return ASTERISK }
func (ga *GenericAsterisk) CompareTo(o c.Comparable) int8 { 
	return CompareGrammarParticles(ga,o.(GrammarParticle))
}

func OpenGrammarBuilder() *GrammarBuilder {
	gb := &GrammarBuilder{
		nonterms: make(map[string]GrammarParticle),
		terms: make(map[string]GrammarParticle),
		rules: tree.NewTree(),
	}
	gb.epsilon = &GenericEpsilon{}
	gb.asterisk = &GenericAsterisk{}
	gb.bottom = &GenericBottom{}
	return gb
}

func (gb *GrammarBuilder) Name(grammarName string) *GrammarBuilder {
	if gb.name != "" {
		panic("Name() called twice")
	}
	gb.name = grammarName
	return gb
}

func (gb *GrammarBuilder) Terminals(termNames ...string) *GrammarBuilder {
	for _, name := range termNames {
		if name[0] == '`' {
			panic("illegal terminal name (may not start with '`') in Terminals()")
		}
		if _, has := gb.nonterms[name]; has {
			panic("nonterminal named '"+name+"' already exists in Terminals()")
		}
		if _, has := gb.terms[name]; has {
			panic("terminal named '"+name+"' already exists in Terminals()")
		}
		gb.terms[name] = &GenericTerminal{name: name}
	}
	return gb
} 

func (gb *GrammarBuilder) Nonterminals(nontermNames ...string) *GrammarBuilder {		

	for _, name := range nontermNames {	
		if name[0] == '`' {
			panic("illegal nonterminal name (may not start with '`') in Nonterminals()")
		}
		if _, has := gb.nonterms[name]; has {
			panic("nonterminal named '"+name+"' already exists in Nonterminals()")
		}
		if _, has := gb.terms[name]; has {
			panic("terminal named '"+name+"' already exists in Nonterminals()")
		}
		gb.nonterms[name] = &GenericNonterminal{name: name}
	}
	return gb
}

func (gb *GrammarBuilder) Rule() *GrammarBuilder {
	if gb.ruleOpen {
		panic("Rule() called twice")
	}
	gb.ruleOpen = true
	gb.lhs = nil
	return gb
}

func (gb *GrammarBuilder) Lhs(lhs ...string) *GrammarBuilder {
	var ok bool
	if !gb.ruleOpen {
		panic("Lhs() called before Rule()")
	}
	if gb.lhs != nil {
		panic("Lhs() called twice")
	}
	if len(lhs) == 0 {
		panic("Lhs() may not have an empty argument list")
	}
	lhsarr := make([]GrammarParticle, 0, len(lhs))
	for _, partName := range lhs {
		var part GrammarParticle
		if partName == "`*" {
			if gb.initialRule != nil {
				panic("initial rule `* -> ... already given by a Rule()")
			}
			if len(lhs) != 1 {
				panic("intial rule `* -> ... must have only one Lhs() argument")
			}
			part = gb.asterisk
		} else {
			if part, ok = gb.nonterms[partName]; !ok {
				if part, ok = gb.terms[partName]; !ok {
					panic("unknown grammar particle '"+partName+"' in Lhs()")
				}
			}
		}
		lhsarr = append(lhsarr, part)
	}
	gb.lhs = lhsarr
	return gb
}

func (gb *GrammarBuilder) Rhs(rhs ...string) *GrammarBuilder {
	var ok bool
	if !gb.ruleOpen {
		panic("Rhs() called before Rule()")
	}
	if gb.lhs == nil {
		panic("Rhs() called before Lhs()")
	}
	if len(rhs) == 0 {
		panic("Rhs() may not have an empty argument list")
	}
	rhsarr := make([]GrammarParticle, 0, len(rhs))
	for _, partName := range rhs {
		var part GrammarParticle
		if partName[0] == '`' {
			if partName == "`e" {
				if len(rhs) > 1 {
					panic("Rhs() with epsilon must be of unit length")
				}
				gb.usedEpsilon = true
				part = gb.epsilon
			} 
			if partName == "`." {
				if !gb.lhs[0].Asterisk() {
					panic("`. may only appear in `* -> ... ")
				}
				part = gb.bottom
			} 
		} 
		if part == nil {
			if part, ok = gb.nonterms[partName]; !ok {
				if part, ok = gb.terms[partName]; !ok {
					panic("unknown grammar particle '"+partName+"' in Rhs()")
				}
			}
		}
		rhsarr = append(rhsarr, part)
	}
	rule := &BasicProduction{
		lhs: gb.lhs,
		rhs: rhsarr,
	}
	gb.ruleOpen = false
	gb.lhs = nil
	if gb.rules.Has(rule) {
		panic("rule already exists during Rhs()")
	}
	if rule.Lhs(0) == gb.asterisk {
		if rule.RhsLen() != 2 {
			panic(rule.Rhs(0).String())
			panic("initial rule `* -> ... must have a Rhs() with exactly two arguments")
		}
		if !rule.Rhs(0).Nonterminal() {
			panic("initial rule `* -> ... must have a Rhs() with a nonterminal argument")
		}
		if !rule.Rhs(1).Bottom() {
			panic("initial rule `* -> ... must have a Rhs() ending with `.")
		}
		gb.initialRule = rule
	}
	gb.rules.Insert(rule)
	return gb	
}

func (gb *GrammarBuilder) Build() (*BasicGrammar,error) {
	if gb.name == "" {
		return nil, errors.New("Name() not called")
	}
	if gb.ruleOpen {
		return nil, errors.New("Rhs() not called after Rule()")
	}
	if gb.initialRule == nil {
		return nil, errors.New("No initial production `* -> ... given")
	}
	var k int
	if gb.usedEpsilon {
		k = 1
	}
	g := &BasicGrammar{
		name: gb.name,
		nonterminals: make([]GrammarParticle, len(gb.nonterms)+1),
		terminals: make([]GrammarParticle, len(gb.terms)+k),
		productions: make([]Production, gb.rules.Size()),
	}
	g.epsilon = &GenericEpsilon{grammar: g}
	g.asterisk = &GenericAsterisk{grammar: g}
	g.bottom = &GenericBottom{grammar: g}
	smap := make(map[GrammarParticle]GrammarParticle)
	smap[gb.asterisk] = g.asterisk
	smap[gb.epsilon] = g.epsilon
	smap[gb.bottom] = g.bottom
	i := 0
	for _, p := range gb.nonterms {
		g.nonterminals[i+1] = &GenericNonterminal{
			name: p.Name(),
			grammar: g,
		}
		smap[p] = g.nonterminals[i+1]
		i++
	}	
	g.nonterminals[0] = g.asterisk
	i = 0
	for _, p := range gb.terms {
		g.terminals[i+k] = &GenericTerminal{
			name: p.Name(),
			grammar: g,
		}
		smap[p] = g.terminals[i+k]
		i++
	}
	if gb.usedEpsilon {
		g.terminals[0] = g.epsilon
	}
	i = 0
	for c := gb.rules.First(); c.HasNext(); {
		g.productions[i], _ = c.Next().(Production).Substitute(smap)
		i++
	}
	return g, nil
}

func (bg *BasicGrammar)	Name() string {
	return bg.name
}

func (bg *BasicGrammar) NumNonterminals() int {
	return len(bg.nonterminals)
}

func (bg *BasicGrammar) Nonterminal(idx int) GrammarParticle {
	return bg.nonterminals[idx]
}

func (bg *BasicGrammar)Nonterminals() []GrammarParticle {
	nts := make([]GrammarParticle, len(bg.nonterminals))
	copy(nts, bg.nonterminals)
	return nts
}

func (bg *BasicGrammar) NumTerminals() int {
	return len(bg.terminals)
}

func (bg *BasicGrammar) Terminal(idx int) GrammarParticle {
	return bg.terminals[idx]
}

func (bg *BasicGrammar) Terminals() []GrammarParticle {
	ts := make([]GrammarParticle, len(bg.terminals))
	copy(ts, bg.terminals)
	return ts
}

func (bg *BasicGrammar) Epsilon() GrammarParticle {
	return bg.epsilon
}

func (bg *BasicGrammar) Asterisk() GrammarParticle {
	return bg.asterisk
}

func (bg *BasicGrammar) Bottom() GrammarParticle {
	return bg.bottom
}

func (bg *BasicGrammar) NumProductions() int {
	return len(bg.productions)
}

func (bg *BasicGrammar) Production(idx int) Production {
	return bg.productions[idx]
}

func (bg *BasicGrammar) Productions() []Production {
	ps := make([]Production, len(bg.productions))
	copy(ps, bg.productions)
	return ps
}

func GetProductionRegularity(p Production) Regularity {
	if p.LhsLen() != 1 || !p.Lhs(0).Nonterminal() {
		return NONREGULAR
	}
	if p.RhsLen() == 1 {
		return STRICT_UNITARY
	}
	if p.RhsLen() == 2 && p.Rhs(0).Terminal() && p.Rhs(1).Nonterminal() {
		return STRICT_RIGHT
	}
	if p.RhsLen() == 2 && p.Rhs(0).Nonterminal() && p.Rhs(1).Terminal() {
		return STRICT_LEFT
	}
	if p.Rhs(0).Nonterminal() {
		for i := 1; i < p.RhsLen(); i++ {
			if p.Rhs(i).Nonterminal() {
				return NONREGULAR
			}
		}
		return LEFT
	}
	if p.Rhs(p.RhsLen()-1).Nonterminal() {
		for i := p.RhsLen()-2; i >= 0; i-- {
			if p.Rhs(i).Nonterminal() {
				return NONREGULAR
			}
		}
		return RIGHT
	}
	for i := 0; i < p.RhsLen(); i++ {
		if p.Rhs(i).Nonterminal() {
			return NONREGULAR
		}
	}
	return UNITARY
}

func GetProductionClass(p Production) ProductionClass {
	//fmt.Println(p.String())
	if p.LhsLen() == 1 && p.Lhs(0).Nonterminal() {
		if p.RhsLen() == 1 && p.Rhs(0).Terminal() {
			return CONSTANT	
		}
		reg := GetProductionRegularity(p)
		if reg.Regular() {
			return REGULAR
		}
		return CONTEXT_FREE
	}
	ntc := 0
	var pos int
	for i := 1; i < p.LhsLen(); i++ {
		if p.Lhs(i).Nonterminal() {
			ntc++
			pos = i
			if ntc == 2 {
				return GENERAL
			}
		}
	}
	if p.RhsLen() < pos {
		return GENERAL
	}
	for i := 0; i < pos; i++ {
		if p.Rhs(i).CompareTo(p.Lhs(i)) != 0 {
			return GENERAL
		}
	}
	for i := 0; i < p.LhsLen()-pos; i++ {
		if p.Lhs(p.LhsLen()-i-1).CompareTo(p.Rhs(p.RhsLen()-i-1)) != 0 {
			return GENERAL
		}
	}
	return CONTEXT_SENSITIVE
}

