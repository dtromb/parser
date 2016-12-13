package lr0

import (
	"fmt"
	"errors"
	"sort"
	"strconv"
	"github.com/dtromb/parser"
	"github.com/dtromb/parser/index"
	c "github.com/dtromb/collections"
	"github.com/dtromb/collections/tree"
)


type LR0Item struct {
	p parser.Production
	pos int
}

type LR0KeyIndex []parser.GrammarParticle

type LR0State struct {
	id int
	items tree.Tree // Tree of LR0Item
	transitions map[parser.GrammarParticle]*LR0State
	reductions map[parser.GrammarParticle][]parser.Production
	keys LR0KeyIndex
}

type LR0Dfa struct {
	states []*LR0State
	terminals []*LR0State
}

func (li *LR0Item) Production() parser.Production {
	return li.p
}

func (li *LR0Item) Position() int {
	return li.pos
}

func (li *LR0Item) Valid() bool {
	return  li.p != nil && li.p.LhsLen() == 1 && 
			(li.p.Lhs(0).Nonterminal() || li.p.Lhs(0).Asterisk()) && 
	        li.pos >= 0 && li.pos <= li.p.RhsLen()
}

func (li *LR0Item) String() string {
	if (!li.Valid()) {
		return "<invalid lr0 item>"
	}
	var buf []byte 
	buf = append(buf, fmt.Sprintf("%s -> ",li.p.Lhs(0).String())...)
	for i := 0; i < li.p.RhsLen(); i++ {
		rt := li.p.Rhs(i)
		if li.pos == i {
			buf = append(buf, ". "...)
		}
		buf = append(buf, fmt.Sprintf("%s ",rt.String())...)
	}
	if li.pos == li.p.RhsLen() {
		buf = append(buf, ". "...)
	}
	//buf = append(buf, strconv.Itoa(li.pos)...)
	return string(buf)
}

func InitialLR0Item(p parser.Production) *LR0Item {
	return &LR0Item{p:p}
}

func (lr *LR0Item) HasNext() bool {
	return lr.pos < lr.p.RhsLen()
}

func (lr *LR0Item) HasPrev() bool {
	return lr.pos > 0
}

func (lr *LR0Item) Next() *LR0Item {
	if lr.pos < lr.p.RhsLen() {
		return &LR0Item{p:lr.p, pos:lr.pos+1}
	}
	return nil
}

func (lr *LR0Item) Prev() *LR0Item {
	if lr.pos > 0 {
		return &LR0Item{p:lr.p, pos:lr.pos-1}
	}
	return nil
}

func (lr *LR0Item) Caret() parser.GrammarParticle {
	if lr.pos < lr.p.RhsLen() {
		return lr.p.Rhs(lr.pos)
	}
	return nil
}
	
func (lra *LR0Item) CompareTo(c c.Comparable) int8 {
	lrb := c.(*LR0Item)
	r := lra.p.CompareTo(lrb.p)
	if r != 0 {
		return r
	}
	if lra.pos > lrb.pos {
		return 1
	}
	if lra.pos < lrb.pos {
		return -1
	}
	return 0
}

func AssignId(lrs *LR0State, id int) {
	lrs.id = id
}

func (lrs *LR0State) Id() int {
	return lrs.id
}

func (lrs *LR0State) String() string {
	var buf []byte
	if lrs == nil {
		return "<nil>"
	}
	if lrs.items == nil || lrs.items.Size() == 0 {
	   buf = append(buf, fmt.Sprintf("[%d: \n   Items: {}\n", lrs.id)...)	
	} else {
		buf = append(buf, fmt.Sprintf("[%d: \n   Items: {\n", lrs.id)...)
		for x := lrs.items.First(); x.HasNext(); {
			buf = append(buf, fmt.Sprintf("      %s\n", x.Next().(*LR0Item))...)
		}
		buf = append(buf, "   }\n"...)
	}
	if lrs.transitions == nil || len(lrs.transitions) == 0 {
		buf = append(buf, fmt.Sprintf("   Transitions: {}\n")...)
	} else {
		buf = append(buf, fmt.Sprintf("   Transitions: {\n")...)
		for p, n := range lrs.transitions {
			buf = append(buf, fmt.Sprintf("      %s => [State %d]\n", p, n.id)...)
		}
		buf = append(buf, "   }\n"...)
	}
	if lrs.reductions == nil || len(lrs.reductions) == 0 {
		buf = append(buf,fmt.Sprintf("   Reductions: {}\n")...)
	} else {
		buf = append(buf, fmt.Sprintf("   Reductions: {\n")...)
		for p, n := range lrs.reductions {
			if len(n) == 1 {
				buf = append(buf, fmt.Sprintf("      %s => %s\n", p, n[0].String())...)
			} else {
				buf = append(buf, fmt.Sprintf("      %s => {\n", p)...)
				for i := 0; i < len(n); i++ {
					buf = append(buf, fmt.Sprintf("         %s\n", n[i].String())...)
				}
				buf = append(buf, "      }\n"...)
			}
		}
		buf = append(buf, "   }\n"...)
	}
	buf = append(buf, "]\n"...)
	return string(buf)
}

func GetTransitions(lrs *LR0State) map[parser.GrammarParticle]*LR0State {
	return lrs.transitions
}

func (lrsa *LR0State) CompareTo(c c.Comparable) int8 {
	lrsb := c.(*LR0State)
	if lrsa.items.Size() > lrsb.items.Size() {
		return 1
	}
	if lrsa.items.Size() < lrsb.items.Size() {
		return -1
	}
	ca := lrsa.items.First()
	cb := lrsb.items.First()
	for ca.HasNext() {
		a := ca.Next()
		b := cb.Next()
		r := a.CompareTo(b)
		if r != 0 {
			return r
		}
	}
	return 0
}

func SeedState(id int, p parser.Production) *LR0State {
	state := LR0State{id:id, items:tree.NewTree(), transitions: make(map[parser.GrammarParticle]*LR0State)}
	state.items.Insert(InitialLR0Item(p))
	return &state
}

func BuildDfa(g parser.Grammar, passEpsilons bool) (*LR0Dfa,error) {
	
	nextId := 1
	
	ig := parser.GetIndexedGrammar(g)
	idx, err := ig.GetIndex(index.GRAMMAR_CLASS_INDEX)
	if err != nil {
		return nil, err
	}
	gcidx := idx.(*index.GrammarClassIndex)
	if !gcidx.Class().ContextFree() {
		return nil, errors.New("cannot create lr0 dfa for a non-context-free grammar")
	}
	idx, err = ig.GetIndex(index.BASIC_INDEX)
	if err != nil {
		return nil, err
	}
	bidx := idx.(*index.BasicGrammarIndex)
	
	canon := tree.NewTree()
	
	start := bidx.LhsStart(ig.Asterisk(),0)
	initState := &LR0State{id:0, items:tree.NewTree(), transitions:make(map[parser.GrammarParticle]*LR0State)}
	initState.items.Insert(InitialLR0Item(start))
	CloseLR0State(initState, ig, passEpsilons)
	canon.Insert(initState)
	newStates := tree.NewTree()
	newStates.Insert(initState)
	for newStates.Size() > 0 {
		ns := newStates.First().Next().(*LR0State)
		newStates.Delete(ns)
		if MakeTransitions(ns, ig) != nil {
			return nil, err
		}
		for part, next := range ns.transitions {
			CloseLR0State(next, ig, passEpsilons)
			if cn, has := canon.Lookup(c.LTE, next); has {
				next = cn.(*LR0State)
			} else {
				next.id = nextId
				nextId++
				canon.Insert(next)
				newStates.Insert(next)
			}
			ns.transitions[part] = next
		}
	}
	
	dfa := &LR0Dfa{}
	dfa.states = make([]*LR0State, nextId)
	for x := canon.First(); x.HasNext(); {
		sx := x.Next().(*LR0State)
		dfa.states[sx.id] = sx
	}
	return dfa, nil
}


// XXX - This should have an option to forward over nihilistic carets.
func MakeTransitions(ci *LR0State, g *parser.IndexedGrammar) error {
	fmt.Println("MAKE TRS: "+ci.String())
	tmap := make(map[parser.GrammarParticle]*LR0State)
	rmap := make(map[parser.GrammarParticle]map[parser.Production]bool)
	for x := ci.items.First(); x.HasNext(); {
		xi := x.Next().(*LR0Item)
		if !xi.HasNext() {
			nt := xi.Production().Lhs(0)
			if _, has := rmap[nt]; !has {
				rmap[nt] = make(map[parser.Production]bool)
			}
			rmap[nt][xi.Production()] = true
		} else {
			tt := xi.Caret()
			if _, has := tmap[tt]; !has {
				tmap[tt] = &LR0State{items:tree.NewTree()}
			} 
			tmap[tt].items.Insert(xi.Next())
		}
	}
	if ci.transitions == nil {
		ci.transitions = tmap
	} else {
		for k, v := range tmap {
			ci.transitions[k] = v
		}
	}
	if ci.reductions == nil {
		ci.reductions = make(map[parser.GrammarParticle][]parser.Production)
	}
	for part, pmap := range rmap {
		if _, has := ci.reductions[part]; !has {
			ci.reductions[part] = make([]parser.Production, 0, len(pmap))
		}
		for prod, _ := range pmap {
			ci.reductions[part] = append(ci.reductions[part], prod)
		}
	}
	return nil
}

func CloseLR0State(s *LR0State, g *parser.IndexedGrammar, forwardEpsilons bool) (bool,error) {
	var changed bool
	fmt.Println("CLOSE: "+s.String())
	idx, err := g.GetIndex(index.BASIC_INDEX)
	if err != nil {
		return false, err
	}
	seennt := tree.NewTree()
	bidx := idx.(*index.BasicGrammarIndex)
	newItems := make([]*LR0Item, 0, s.items.Size())
	for cr := s.items.First(); cr.HasNext(); {
		newItems = append(newItems, cr.Next().(*LR0Item))
	}
	for len(newItems) > 0 {
		ci := newItems[len(newItems)-1]
		newItems = newItems[0:len(newItems)-1]
		if _, has := s.items.Lookup(c.LTE, ci); !has {
			s.items.Insert(ci)
			changed = true
		}
		if !ci.HasNext() {
			continue
		}
		at := ci.Caret()
		if !(at.Nonterminal() || at.Asterisk()) {
			continue
		}
		if forwardEpsilons && bidx.Epsilon(at) {
			ni := ci.Next()
			if !ni.HasNext() {
				// spurious epsilon reduction
				nt := ni.Production().Lhs(0)
				if s.reductions == nil {
					s.reductions = make(map[parser.GrammarParticle][]parser.Production)
				}
				if _, has := s.reductions[nt]; !has {
					s.reductions[nt] = make([]parser.Production,0,1)
				}
				var has bool 
				for _, k := range s.reductions[nt] {
					if k.CompareTo(ni.Production()) == 0 {
						has = true
						break
					}
				}
				if !has {
					s.reductions[nt] = append(s.reductions[nt], ni.Production())
				}
			}
			newItems = append(newItems, ni)
		}
		if _, has := seennt.Lookup(c.LTE, at); has {
			continue
		}
		seennt.Insert(at)
		for i := 0; i < bidx.NumLhsStarts(at); i++ {
			newItems = append(newItems, InitialLR0Item(bidx.LhsStart(at, i)))
		}
	}
	return changed, nil
}

func SetTransition(from *LR0State, to *LR0State, on parser.GrammarParticle) {
	if from.transitions == nil {
		from.transitions = make(map[parser.GrammarParticle]*LR0State)
	}
	from.transitions[on] = to
}

type Lr0ItemFilterFn func(item *LR0Item) bool
type Lr0ItemTransformFn func(item *LR0Item) *LR0Item

func SplitLR0State(newid int, state *LR0State, filter Lr0ItemFilterFn, transform Lr0ItemTransformFn) *LR0State {
	filterItems := tree.NewTree()
	addItems := tree.NewTree()
	for x := state.items.First(); x.HasNext(); {
		it := x.Next().(*LR0Item)
		if filter(it) {
			filterItems.Insert(it)
		}
		ni := transform(it)
		if ni != nil {
			addItems.Insert(ni)
		}
	}
	if filterItems.Size() == 0 && addItems.Size() == 0 {
		return nil
	}
	for x := filterItems.First(); x.HasNext(); {
		state.items.Delete(x.Next().(*LR0Item))
	}
	for x := addItems.First(); x.HasNext(); {
		filterItems.Insert(x.Next().(*LR0Item))
	}
	newState := &LR0State{id:newid, items: filterItems}
	return newState
}

func (lr *LR0State) Label() string {
	return strconv.Itoa(lr.id)
}

func (lr *LR0State) HasTransition(k c.Comparable) bool {
	gp, ok := k.(parser.GrammarParticle)
	if !ok {
		return false
	}
	_, has := lr.transitions[gp]
	return has
}

func (lr *LR0State) Transition(k c.Comparable) parser.DfaState {
	gp, ok := k.(parser.GrammarParticle)
	if !ok {
		return nil
	}
	return lr.transitions[gp]
}

func (lr *LR0State) Terminal() bool {
	return len(lr.transitions) == 0
}

func (lr *LR0State) NumTransitions() int {
	return len(lr.transitions)
}

func (lr *LR0State) TransitionKey(idx int) c.Comparable {
	if lr.keys == nil {
		lr.keys = make([]parser.GrammarParticle,0,len(lr.transitions))
		for k, _ := range lr.transitions {
			lr.keys = append(lr.keys, k)
		}
		sort.Sort(lr.keys)
	}
	if idx < 0 || idx >= len(lr.keys) {
		return nil
	}
	return lr.keys[idx]
}

func (lr *LR0Dfa) InitialState() parser.DfaState {
	return lr.states[0]
}

func (lr *LR0Dfa) NumStates() int {
	return len(lr.states)
}

func (lr *LR0Dfa) State(idx int) parser.DfaState {
	if idx < 0 || idx >= len(lr.states) {
		return nil
	}
	return lr.states[idx]
}

func (lr *LR0Dfa) NumTerminals() int {
	if lr.terminals == nil {
		lr.terminals = []*LR0State{}
		for _, s := range lr.states {
			if s.Terminal() {
				lr.terminals = append(lr.terminals, s)
			}
		}
	}
	return len(lr.terminals)
}

func (lr *LR0Dfa) Terminal(idx int) parser.DfaState {
	if idx < 0 || idx >= lr.NumTerminals() {
		return nil
	}
	return lr.terminals[idx]
}

func (ls *LR0State) FirstItem() c.Cursor {
	return ls.items.First()
}

func (ki LR0KeyIndex) Len() int {
	return len([]parser.GrammarParticle(ki))
}

func (ki LR0KeyIndex) Less(i, j int) bool {
	s := []parser.GrammarParticle(ki)
	return s[i].CompareTo(s[j]) < 0
}

func (ki LR0KeyIndex) Swap(i, j int) {
	s := []parser.GrammarParticle(ki)
	s[i], s[j] = s[j], s[i]
}    