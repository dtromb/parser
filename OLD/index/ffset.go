package index

import (
	"errors"
	"reflect"
	c "github.com/dtromb/collections"
	"github.com/dtromb/collections/tree"
	"github.com/dtromb/parser"
)

// First and follow (and inset) set indexing.

type FFIndex struct {
	grammar parser.Grammar
	firstSets map[parser.GrammarParticle][]parser.GrammarParticle
	lastSets map[parser.GrammarParticle][]parser.GrammarParticle
	inSets map[parser.GrammarParticle][]parser.GrammarParticle
	followSets map[parser.GrammarParticle][]parser.GrammarParticle
	precedeSets map[parser.GrammarParticle][]parser.GrammarParticle
}

const(
	FIRST_FOLLOW_INDEX	string = "first-follow"
)

func init() {
	parser.RegisterGrammarIndexType(reflect.TypeOf([]FFIndex{}).Elem())
}

func (ff *FFIndex) IndexName() string {
	return FIRST_FOLLOW_INDEX
}

func (ff *FFIndex) Grammar() parser.Grammar {
	return ff.grammar
}

func (ff *FFIndex) Initialize(g parser.Grammar) error {
	ff.grammar = g
	index := parser.GetIndexedGrammar(g)
	idx, err := index.GetIndex(GRAMMAR_CLASS_INDEX)
	if err != nil {
		return err
	}
	cidx := idx.(*GrammarClassIndex)
	if cidx.Class() >= parser.CONTEXT_SENSITIVE {
		return errors.New("cannot first/follow index a non-context-free grammar")
	}
	idx, err = index.GetIndex(BASIC_INDEX)
	bidx := idx.(*BasicGrammarIndex)
	if err != nil {
		return err
	}
	
	// FIRST set calculation
	ff.firstSets = make(map[parser.GrammarParticle][]parser.GrammarParticle)
	for _, nt := range index.Nonterminals() {
		fs := tree.NewTree()
		ntseen := tree.NewTree()
		ntpending := []parser.GrammarParticle{nt}
		for len(ntpending) > 0 {
			cnt := ntpending[0]
			ntpending = ntpending[1:]
			for i := 0; i < bidx.NumLhsStarts(cnt); i++ {
				p := bidx.LhsStart(cnt,i)
				for j := 0; j < p.RhsLen(); j++ {
					rt := p.Rhs(j)
					if rt.Terminal() {
						fs.Insert(rt)
						break
					} else if rt.Nonterminal() {
						if _, has := ntseen.Lookup(c.LTE, rt); !has {
							ntseen.Insert(rt)
							fs.Insert(rt)
							ntpending = append(ntpending,rt)
						}
						if !bidx.Epsilon(rt) {
							break
						} 
					} else {
						break	
					}
				}
			}
		}
		ff.firstSets[nt] = make([]parser.GrammarParticle, 0, fs.Size())
		for c := fs.First(); c.HasNext(); {
			ff.firstSets[nt] = append(ff.firstSets[nt], c.Next().(parser.GrammarParticle))
		}
	}
	
	// LAST set calculation
	ff.lastSets = make(map[parser.GrammarParticle][]parser.GrammarParticle)
	for _, nt := range index.Nonterminals() {
		fs := tree.NewTree()
		ntseen := tree.NewTree()
		ntpending := []parser.GrammarParticle{nt}
		for len(ntpending) > 0 {
			cnt := ntpending[0]
			ntpending = ntpending[1:]
			for i := 0; i < bidx.NumLhsStarts(cnt); i++ {
				p := bidx.LhsStart(cnt,i)
				for j := p.RhsLen()-1; j >= 0; j-- {
					rt := p.Rhs(j)
					if rt.Terminal() {
						fs.Insert(rt)
						break
					}
					if rt.Nonterminal() {
						if _, has := ntseen.Lookup(c.LTE, rt); !has {
							ntseen.Insert(rt)
							fs.Insert(rt)
							ntpending = append(ntpending,rt)
							if !bidx.Epsilon(rt) {
								break
							}
						}
					}
				}
			}
		}
		ff.lastSets[nt] = make([]parser.GrammarParticle, 0, fs.Size())
		for c := fs.First(); c.HasNext(); {
			ff.lastSets[nt] = append(ff.lastSets[nt], c.Next().(parser.GrammarParticle))
		}
	}
	
	// IN set calculation
	ff.inSets = make(map[parser.GrammarParticle][]parser.GrammarParticle)
	for _, nt := range index.Nonterminals() {
		fs := tree.NewTree()
		ntseen := tree.NewTree()
		ntpending := []parser.GrammarParticle{nt}
		for len(ntpending) > 0 {
			cnt := ntpending[0]
			ntpending = ntpending[1:]
			for i := 0; i < bidx.NumLhsStarts(cnt); i++ {
				p := bidx.LhsStart(cnt,i)
				for j := p.RhsLen()-1; j >= 0; j-- {
					rt := p.Rhs(j)
					if rt.Terminal() {
						fs.Insert(rt)
					}
					if rt.Nonterminal() {
						if _, has := ntseen.Lookup(c.LTE, rt); !has {
							ntseen.Insert(rt)
							fs.Insert(rt)
							ntpending = append(ntpending,rt)
						}
					}
				}
			}
		}
		ff.inSets[nt] = make([]parser.GrammarParticle, 0, fs.Size())
		for c := fs.First(); c.HasNext(); {
			ff.inSets[nt] = append(ff.inSets[nt], c.Next().(parser.GrammarParticle))
		}
	}
	
	
	// FOLLOW set calculation
	followRefs := make(map[parser.GrammarParticle]tree.Tree)
	followSets := make(map[parser.GrammarParticle]tree.Tree)
	for _, p := range g.Productions() { // First-pass.
		for i := 0; i < p.RhsLen()-1; i++ {
			for j := i+1; j < p.RhsLen(); j++ {
				if _, has := followSets[p.Rhs(i)]; !has {
					followSets[p.Rhs(i)] = tree.NewTree()
				}
				followSets[p.Rhs(i)].Insert(p.Rhs(j))
				if !bidx.Epsilon(p.Rhs(j)) {
					break
				}
			}
		}
		tp := p.Rhs(p.RhsLen()-1)
		if _, has := followRefs[tp]; !has {
			followRefs[tp] = tree.NewTree()
		}
		followRefs[tp].Insert(p.Lhs(0))
	}
	var changed bool = true
	for changed { // Take closure.
		changed = false
		for p, prt := range followRefs {
			for cr := prt.First(); cr.HasNext(); {
				fp := cr.Next().(parser.GrammarParticle) // x in Follow(fp) -> x in Follow(p)
				if fromSet, has := followSets[fp]; has {
					if _, has := followSets[p]; !has {
						followSets[p] = tree.NewTree()
					}
					for k := fromSet.First(); k.HasNext(); {
						x := k.Next().(parser.GrammarParticle)
						if _, has := followSets[p].Lookup(c.LTE, x); !has {
							changed = true
							followSets[p].Insert(x)
						}
					}
				}
			}
		}
	}
	ff.followSets = make(map[parser.GrammarParticle][]parser.GrammarParticle)
	for r, v := range followSets { // Collect results.
		ff.followSets[r] = make([]parser.GrammarParticle, 0, v.Size())
		for c := v.First(); c.HasNext(); {
			ff.followSets[r] = append(ff.followSets[r], c.Next().(parser.GrammarParticle))
		}
	}
	
	return nil
}

func (ff *FFIndex) NumFirsts(p parser.GrammarParticle) int {
	if _, has := ff.firstSets[p]; has {
		return len(ff.firstSets[p])
	}
	return 0
}

func (ff *FFIndex) First(p parser.GrammarParticle, idx int) parser.GrammarParticle {
	if _, has := ff.firstSets[p]; has {
		return ff.firstSets[p][idx]
	}
	return nil
}

func (ff *FFIndex) Firsts(p parser.GrammarParticle) []parser.GrammarParticle {
	if _, has := ff.firstSets[p]; has {
		rv := make([]parser.GrammarParticle, len(ff.firstSets[p]))
		copy(rv, ff.firstSets[p])
		return rv
	}
	return []parser.GrammarParticle{}
}

func (ff *FFIndex) NumLasts(p parser.GrammarParticle) int {
	if _, has := ff.lastSets[p]; has {
		return len(ff.lastSets[p])
	}
	return 0
}

func (ff *FFIndex) Last(p parser.GrammarParticle, idx int) parser.GrammarParticle {
	if _, has := ff.lastSets[p]; has {
		return ff.lastSets[p][idx]
	}
	return nil
}

func (ff *FFIndex) Lasts(p parser.GrammarParticle) []parser.GrammarParticle {
	if _, has := ff.lastSets[p]; has {
		rv := make([]parser.GrammarParticle, len(ff.lastSets[p]))
		copy(rv, ff.lastSets[p])
		return rv
	}
	return []parser.GrammarParticle{}
}

func (ff *FFIndex) NumIns(p parser.GrammarParticle) int {
	if _, has := ff.inSets[p]; has {
		return len(ff.inSets[p])
	}
	return 0
}

func (ff *FFIndex) In(p parser.GrammarParticle, idx int) parser.GrammarParticle {
	if _, has := ff.inSets[p]; has {
		return ff.inSets[p][idx]
	}
	return nil
}

func (ff *FFIndex) Ins(p parser.GrammarParticle) []parser.GrammarParticle {
	if _, has := ff.inSets[p]; has {
		rv := make([]parser.GrammarParticle, len(ff.inSets[p]))
		copy(rv, ff.inSets[p])
		return rv
	}
	return []parser.GrammarParticle{}
}

func (ff *FFIndex) NumFollows(p parser.GrammarParticle) int {
	if _, has := ff.followSets[p]; has {
		return len(ff.followSets[p])
	}
	return 0
}

func (ff *FFIndex) Follow(p parser.GrammarParticle, idx int) parser.GrammarParticle {
	if _, has := ff.followSets[p]; has {
		return ff.followSets[p][idx]
	}
	return nil
}

func (ff *FFIndex) Follows(p parser.GrammarParticle) []parser.GrammarParticle {
	if _, has := ff.followSets[p]; has {
		rv := make([]parser.GrammarParticle, len(ff.followSets[p]))
		copy(rv, ff.followSets[p])
		return rv
	}
	return []parser.GrammarParticle{}
}

func (ff *FFIndex) NumLeads(p parser.GrammarParticle) int {
	if _, has := ff.precedeSets[p]; has {
		return len(ff.precedeSets[p])
	}
	return 0
}

func (ff *FFIndex) Lead(p parser.GrammarParticle, idx int) parser.GrammarParticle {
	if _, has := ff.precedeSets[p]; has {
		return ff.precedeSets[p][idx]
	}
	return nil
}

func (ff *FFIndex) Leads(p parser.GrammarParticle) []parser.GrammarParticle {
	if _, has := ff.precedeSets[p]; has {
		rv := make([]parser.GrammarParticle, len(ff.precedeSets[p]))
		copy(rv, ff.precedeSets[p])
		return rv
	}
	return []parser.GrammarParticle{}
}