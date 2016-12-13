package index

import (
	"github.com/dtromb/parser"
	"reflect"
)

// Basic grammar index provides left/right inclusion indexes for
// productions.

type BasicGrammarIndex struct {
	grammar parser.Grammar
	lhsIncludes map[parser.GrammarParticle][]parser.Production
	rhsIncludes map[parser.GrammarParticle][]parser.Production
	lhsStarts map[parser.GrammarParticle][]parser.Production
	rhsStarts map[parser.GrammarParticle][]parser.Production
	lhsEnds map[parser.GrammarParticle][]parser.Production
	rhsEnds map[parser.GrammarParticle][]parser.Production
	epsilons map[parser.GrammarParticle]parser.Production
}

const(
	BASIC_INDEX	string = "basic"
)

func init() {
	parser.RegisterGrammarIndexType(reflect.TypeOf([]BasicGrammarIndex{}).Elem())
}

func gpSliceCopy(k []parser.Production) []parser.Production {
	s := make([]parser.Production, len(k))
	copy(s,k)
	return s
}

func (bgi *BasicGrammarIndex) Epsilon(p parser.GrammarParticle) bool {
	_, has := bgi.epsilons[p]
	return has
} 

func (bgi *BasicGrammarIndex) EpsilonProduction(p parser.GrammarParticle) parser.Production {
	return bgi.epsilons[p]
}

func (bgi *BasicGrammarIndex) NumLhsStarts(p parser.GrammarParticle) int {
	return len(bgi.lhsStarts[p])
}
func (bgi *BasicGrammarIndex) LhsStart(p parser.GrammarParticle, idx int) parser.Production {
	return bgi.lhsStarts[p][idx]
}
func (bgi *BasicGrammarIndex) LhsStarts(p parser.GrammarParticle) []parser.Production {
	return gpSliceCopy(bgi.lhsStarts[p])
}





func (bgi *BasicGrammarIndex) NumRhsStarts(p parser.GrammarParticle) int {
	return len(bgi.rhsStarts[p])
}
func (bgi *BasicGrammarIndex) RhsStart(p parser.GrammarParticle, idx int) parser.Production {
	return bgi.rhsStarts[p][idx]
}
func (bgi *BasicGrammarIndex) RhsStarts(p parser.GrammarParticle) []parser.Production {
	return gpSliceCopy(bgi.rhsStarts[p])
}



func (bgi *BasicGrammarIndex) NumLhsEnds(p parser.GrammarParticle) int {
	return len(bgi.lhsEnds[p])
}
func (bgi *BasicGrammarIndex) LhsEnd(p parser.GrammarParticle, idx int) parser.Production {
	return bgi.lhsEnds[p][idx]
}
func (bgi *BasicGrammarIndex) LhsEnds(p parser.GrammarParticle) []parser.Production {
	return gpSliceCopy(bgi.lhsEnds[p])
}

func (bgi *BasicGrammarIndex) NumRhsEnds(p parser.GrammarParticle) int {
	return len(bgi.rhsEnds[p])
}
func (bgi *BasicGrammarIndex) RhsEnd(p parser.GrammarParticle, idx int) parser.Production {
	return bgi.rhsEnds[p][idx]
}
func (bgi *BasicGrammarIndex) RhsEnds(p parser.GrammarParticle) []parser.Production {
	return gpSliceCopy(bgi.rhsEnds[p])	
}

func (bgi *BasicGrammarIndex) NumLhsContains(p parser.GrammarParticle) int {
	return len(bgi.lhsIncludes[p])
}
func (bgi *BasicGrammarIndex) LhsContain(p parser.GrammarParticle, idx int) parser.Production {
	return bgi.lhsIncludes[p][idx]
}
func (bgi *BasicGrammarIndex) LhsContains(p parser.GrammarParticle) []parser.Production {
	return gpSliceCopy(bgi.lhsIncludes[p])
}

func (bgi *BasicGrammarIndex) NumRhsContains(p parser.GrammarParticle) int {
	return len(bgi.rhsIncludes[p])
}
func (bgi *BasicGrammarIndex) RhsContain(p parser.GrammarParticle, idx int) parser.Production {
	return bgi.rhsIncludes[p][idx]
}
func (bgi *BasicGrammarIndex) RhsContains(p parser.GrammarParticle) []parser.Production {
	return gpSliceCopy(bgi.rhsIncludes[p])	
}

func (bgi *BasicGrammarIndex) IndexName() string {
	return BASIC_INDEX
}

func (bgi *BasicGrammarIndex) Grammar() parser.Grammar {
	return bgi.grammar
}

func (bgi *BasicGrammarIndex) Initialize(g parser.Grammar) error {
	bgi.grammar = g
	bgi.epsilons = make(map[parser.GrammarParticle]parser.Production)
	bgi.lhsIncludes = make(map[parser.GrammarParticle][]parser.Production)
	bgi.rhsIncludes = make(map[parser.GrammarParticle][]parser.Production)
	bgi.lhsStarts = make(map[parser.GrammarParticle][]parser.Production)
	bgi.rhsStarts = make(map[parser.GrammarParticle][]parser.Production)
	bgi.lhsEnds = make(map[parser.GrammarParticle][]parser.Production)
	bgi.rhsEnds = make(map[parser.GrammarParticle][]parser.Production)
	lhicn := make(map[parser.GrammarParticle]map[parser.Production]int)
	rhicn := make(map[parser.GrammarParticle]map[parser.Production]int)
	for _, p := range g.Productions() {
		if p.LhsLen() == 1 && p.Lhs(0).Asterisk() {
			bgi.lhsStarts[p.Lhs(0)] = []parser.Production{p}
			bgi.lhsEnds[p.Lhs(0)] = []parser.Production{p}
			bgi.rhsStarts[p.Lhs(0)] = []parser.Production{p}
			if _, has := bgi.rhsStarts[p.Rhs(0)]; !has {
				bgi.rhsStarts[p.Rhs(0)] = []parser.Production{}
			}
			bgi.rhsIncludes[p.Rhs(0)] = append(bgi.rhsIncludes[p.Rhs(0)],p)
			if _, has := bgi.rhsIncludes[p.Rhs(0)]; !has {
				bgi.rhsIncludes[p.Rhs(0)] = []parser.Production{}
			}
			bgi.rhsIncludes[p.Rhs(1)] = []parser.Production{p}
			bgi.rhsEnds[p.Rhs(1)] = []parser.Production{p}
			continue
		}
		if p.LhsLen() == 1 && p.Lhs(0).Nonterminal() && p.RhsLen() == 1 && p.Rhs(0).Epsilon() {
			bgi.epsilons[p.Lhs(0)] = p
		}
		iterm := p.Lhs(0)
		bgi.lhsStarts[iterm] = append(bgi.lhsStarts[iterm], p)
		eterm := p.Lhs(p.LhsLen()-1)
		bgi.lhsEnds[eterm] = append(bgi.lhsEnds[eterm], p) 
		iterm = p.Rhs(0)
		bgi.rhsStarts[iterm] = append(bgi.rhsStarts[iterm], p)
		eterm = p.Rhs(p.RhsLen()-1)
		bgi.rhsEnds[eterm] = append(bgi.rhsEnds[eterm], p)
		for idx := 0; idx < p.LhsLen(); idx++ {
			pt := p.Lhs(idx)
			if m, has := lhicn[pt]; !has {
				m = make(map[parser.Production]int)
				m[p] = 1
				lhicn[pt] = m
			} else {
				lhicn[pt][p] = 1
			}
		}		
		for idx := 0; idx < p.RhsLen(); idx++ {
			pt := p.Rhs(idx)
			if m, has := rhicn[pt]; !has {
				m = make(map[parser.Production]int)
				m[p] = 1
				rhicn[pt] = m
			} else {
				rhicn[pt][p] = 1
			}
		}	
	}
	for pt, set := range lhicn {
		slc := make([]parser.Production, 0, len(set))
		for p, _ := range set {
			slc = append(slc, p)
		}
		bgi.lhsIncludes[pt] = slc
	}
	for pt, set := range rhicn {
		slc := make([]parser.Production, 0, len(set))
		for p, _ := range set {
			slc = append(slc, p)
		}
		bgi.rhsIncludes[pt] = slc
	}
	
	// Close epsilons.
	changed := true
	for changed {
		changed = false
	 	for _, p := range g.Productions() {
			if p.LhsLen() != 1 || !p.Lhs(0).Nonterminal() {
				continue
			}
			nt := p.Lhs(0)
			if bgi.Epsilon(nt) {
				continue
			}
			neweps := true
			for i := 0; i < p.RhsLen(); i++ {
				t := p.Rhs(i)
				if !t.Nonterminal() || !bgi.Epsilon(t) {
					neweps = false
					break
				}
			}
			if neweps {
				bgi.epsilons[nt] = p
				changed = true
			}
		}
	}
	
	return nil
}