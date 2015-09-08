package earley

import (
	"reflect"
	"errors"
	"fmt"
	"github.com/dtromb/parser"
	"github.com/dtromb/parser/lr0"
	"github.com/dtromb/parser/index"
	c "github.com/dtromb/collections"
	"github.com/dtromb/collections/tree"
)

type Elr0Dfa struct{
	states []*lr0.LR0State
	terminals []*lr0.LR0State
}

type Elr0ReducedDfaState struct {
	id int
	reductions map[parser.GrammarParticle][]parser.Production
	transitions map[parser.GrammarParticle]*Elr0ReducedDfaState
}

type Elr0ParseState struct {
	id int
	items []*Elr0ParseItem
	itemsByState map[uint64]*Elr0ParseItem
	//reductions []*Elr0ParseItem // Indexes items with reductions present in their dfastate
	//transitions map[parser.GrammarParticle][]int  // Caches nonterminal transitions from the items
}

type Elr0CausalLink struct {
	cause *Elr0ParseItem
	pred *Elr0ParseItem
}

type Elr0ParseItem struct {
	parseStateId int
	dfaState *Elr0ReducedDfaState
	parentState *Elr0ParseState
	links []*Elr0CausalLink
	particle parser.GrammarParticle
}

type Elr0Parser struct {
	grammar parser.Grammar
	factory parser.SyntaxValueFactory
	states []Elr0ReducedDfaState
	transformChain []parser.SyntaxTreeTransform
}

func advanceIndexSelect(item *lr0.LR0Item) bool {
	return item.Position() == 0 && !item.Production().Lhs(0).Asterisk()
}

func BuildEpsilonLR0Dfa(g parser.Grammar) (*Elr0Dfa,error) {
	ig := parser.GetIndexedGrammar(g)
	idx, err := ig.GetIndex(index.BASIC_INDEX)
	if err != nil {
		return nil, err
	}
	bidx := idx.(*index.BasicGrammarIndex)
	idx, err = ig.GetIndex(index.GRAMMAR_CLASS_INDEX)
	if err != nil {
		return nil, err
	}
	cidx := idx.(*index.GrammarClassIndex)
	if !cidx.Class().ContextFree() {
		return nil, errors.New("cannot build an ε-lr0 dfa for a non-context-free grammar")
	}
		
	advanceIndexForward := func(item *lr0.LR0Item) *lr0.LR0Item {
		if !item.HasNext() {
			return nil
		}
		if bidx.Epsilon(item.Caret()) {
			return item.Next()
		}
		return nil
	}
	
	canon := tree.NewTree()
	newStates := tree.NewTree()
	
	nextId := 1
	start := bidx.LhsStart(ig.Asterisk(),0)
	initState := lr0.SeedState(0, start) 
	lr0.CloseLR0State(initState, ig, false)
	eInit := lr0.SplitLR0State(nextId, initState, advanceIndexSelect, advanceIndexForward)
	if eInit != nil {
		//nextId++  Tweaky juggling... this is the first new id, we want to use #1
		lr0.CloseLR0State(eInit, ig, true)
		lr0.SetTransition(initState, eInit, ig.Epsilon())
	}
	canon.Insert(initState)
	newStates.Insert(initState)
//	if eInit != nil {
//		canon.Insert(eInit)
//		newStates.Insert(eInit)
//	}
	
	fmt.Println("INIT STATE: "+initState.String())
	fmt.Println("INIT-e STATE: "+eInit.String())
	for newStates.Size() > 0 {
		fmt.Printf("NS LEN: %d\n", newStates.Size())
		cs := newStates.First().Next().(*lr0.LR0State)
		_, has := newStates.Delete(cs)
		if !has {
			fmt.Println("LOOKING AT "+cs.String())
			tree.DumpTree(newStates.(*tree.AvlTree))
			panic("FAILED DELETE")
		}
		fmt.Printf("NS LEN after del : %d\n", newStates.Size())
		fmt.Println("TOOK STATE: "+cs.String())
		if lr0.MakeTransitions(cs, ig) != nil {
			return nil, err
		}
		for part, next := range lr0.GetTransitions(cs) {
			fmt.Println("A")
			lr0.CloseLR0State(next, ig, false)
			if nxt, has := canon.Lookup(c.LTE, next); has {
				next = nxt.(*lr0.LR0State)
				lr0.SetTransition(cs, next, part)
				continue
			}
			if !part.Epsilon() {
				fmt.Println("B?")
				eNext := lr0.SplitLR0State(nextId, next, advanceIndexSelect, advanceIndexForward)
				if eNext == nil {
					fmt.Println("B-")
					canon.Insert(next)
					newStates.Insert(next)
					lr0.AssignId(next, nextId)
					nextId++
					continue
				}
				fmt.Println("B+")
				lr0.CloseLR0State(eNext, ig, true)
				if nxt, has := canon.Lookup(c.LTE, eNext); has {
					eNext = nxt.(*lr0.LR0State)
				} else {
					canon.Insert(eNext)
					newStates.Insert(eNext)
					nextId++
				}
				if nxt, has := canon.Lookup(c.LTE, next); has {
					next = nxt.(*lr0.LR0State)
				} else {
					canon.Insert(next)
					newStates.Insert(next)
					lr0.AssignId(next, nextId)
					nextId++
				}
				lr0.SetTransition(next, eNext, ig.Epsilon())
			} else {
				if nxt, has := canon.Lookup(c.LTE, next); has {
					next = nxt.(*lr0.LR0State)
				} else {
					canon.Insert(next)
					newStates.Insert(next)
					lr0.AssignId(next, nextId)
					nextId++
				}
				lr0.SetTransition(cs, next, part)
			}
		}
	}
	dfa := &Elr0Dfa{}
	dfa.states = make([]*lr0.LR0State, nextId)
	for x := canon.First(); x.HasNext(); {
		sx := x.Next().(*lr0.LR0State)
		dfa.states[sx.Id()] = sx
	}
	return dfa, nil
}

func (ed *Elr0Dfa) InitialState() parser.DfaState {
	return ed.states[0]
}

func (ed *Elr0Dfa) NumStates() int {
	return len(ed.states)
}

func (ed *Elr0Dfa) State(idx int) parser.DfaState {
	if idx < 0 || idx >= len(ed.states) {
		return nil
	}
	return ed.states[idx]
}

func (ed *Elr0Dfa) NumTerminals() int {
	if ed.terminals == nil {
		ed.terminals = []*lr0.LR0State{}
		for _, s := range ed.states {
			if s.Terminal() {
				ed.terminals = append(ed.terminals, s)
			}
		}
	}
	return len(ed.terminals)
}

func (ed *Elr0Dfa) Terminal(idx int) parser.DfaState {
	if idx < 0 || idx >= ed.NumTerminals() {
		return nil
	}
	return ed.terminals[idx]
}
	
func GenerateParser(g parser.Grammar) (parser.Parser,error) {
	
	var dfa *Elr0Dfa
	var invT parser.SyntaxTreeTransform
	
	itChain := []parser.SyntaxTreeTransform{}
	nnf, err := IsNihilisticNormalForm(g)
	if err != nil {
		return nil, err
	}
	if !nnf {
		// Transform the grammar to NNF.
		g, invT, err = GetNihilisticAugmentGrammar(g)
		if err != nil {
			return nil, err
		}
		if invT != nil {
			itChain = append(itChain, invT)
		}
	}
	
	// XXX - Check grammar for unreachable terms and productions and prune these.
	
	// Create the dfa.
	dfa, err = BuildEpsilonLR0Dfa(g)
	if err != nil {
		return nil, err
	}
	
	// Reduce the dfa to the minimal internal parser structure.
	states := make([]Elr0ReducedDfaState, dfa.NumStates())
	for i := 0; i < len(states); i++ {
		state := dfa.states[i]
		states[i].id = state.Id()
		for c := state.FirstItem(); c.HasNext(); {
			item := c.Next().(*lr0.LR0Item)
			if !item.HasNext() {
				nt := item.Production().Lhs(0)
				if states[i].reductions == nil {
					states[i].reductions = make(map[parser.GrammarParticle][]parser.Production)
				} 
				if _, has := states[i].reductions[nt]; !has {
					states[i].reductions[nt] = []parser.Production{item.Production()}
				} else {
					states[i].reductions[nt] = append(states[i].reductions[nt], item.Production())
				}
			}
		}
		for j := 0; j < state.NumTransitions(); j++ {
			key := state.TransitionKey(j).(parser.GrammarParticle)
			toid := state.Transition(key).Id()
			if states[i].transitions == nil {	
				states[i].transitions = make(map[parser.GrammarParticle]*Elr0ReducedDfaState)
			} 
			states[i].transitions[key] = &states[toid]
		}
	}
	parser := &Elr0Parser{
		grammar: g,
		states: states,
		transformChain: itChain,
	}
	
	for _, dfaState := range states {
		fmt.Printf("DFA STATE [%d]:\n", dfaState.id)
		for nt, next := range dfaState.transitions {
			fmt.Printf("    %s -> [%d]\n", nt.String(), next.id)
		}
		for nt, rs := range dfaState.reductions {
			for _, r := range rs {
				fmt.Printf("    [%s] : %s\n", nt.String(), r.String())
			}
		}
	}
	return parser, nil
}

type Stringable interface {
	String() string
}

func (p *Elr0Parser) Error(msg string) parser.ParserError {
	panic("unimplemented B: "+msg)
}

func (p *Elr0Parser) ParseFailureError(lastState *Elr0ParseState) parser.ParserError {
	for _, item := range lastState.items {
		fmt.Printf("%d %d\n", item.dfaState.id, item.parentState.id)
	}
	panic("unimplemented C")
}

func (p *Elr0Parser) Parse(lexer parser.Lexer, svf parser.SyntaxValueFactory) (ast parser.SyntaxTreeNode, err parser.ParserError) {
	
	// Store value factory reference XXX - is this the best place to keep this?
	p.factory = svf
	
	// Cache epsilon, we reference it a few times.
	e := p.grammar.Epsilon()
	
	// Number of states used as hash multiplier.
	n := uint64(len(p.states))
	
	// Create the initial state.
	initialState := &Elr0ParseState{
		itemsByState: make(map[uint64]*Elr0ParseItem),
	}
	initialItem := &Elr0ParseItem{
		dfaState: &p.states[0],
		parentState: initialState,
	}
	if enext, has := p.states[0].transitions[e]; has {
		initialEItem := &Elr0ParseItem{
			dfaState: enext,
			parentState: initialState,
		}
		initialState.items = []*Elr0ParseItem{initialItem,initialEItem}
	} else {
		initialState.items = []*Elr0ParseItem{initialItem}
	}
	
	var cs *Elr0ParseState
	ns := initialState
	
	for !lexer.Eof() {
		
		// Read a token.
		var token parser.GrammarParticle
		lexerToken, err := lexer.Next()
		if err != nil {
			return nil, p.Error(err.Error())
		}
		fmt.Printf("\nSTATE: %d, TOKEN: %s\n",ns.id,lexerToken.String())
		// Canonicalize by discarding value.
		if vt, isValue := lexerToken.(*parser.ValueTerminal); isValue {
			token = vt.CanonicalTerminal()
		} else {
			token = lexerToken
		}
		
		// Hold state n-1, and create state n.
		cs = ns
		ns = &Elr0ParseState{
			id: cs.id+1,
			items: []*Elr0ParseItem{},
			itemsByState: make(map[uint64]*Elr0ParseItem),
		}
		
		for i := 0; i < len(cs.items); i++ {
			item := cs.items[i]
			fmt.Printf("   @ (%d) %d %d\n", cs.id, item.dfaState.id, item.parentState.id)
			if next, has := item.dfaState.transitions[token]; has {
				// SCAN step: transition on terminal edge of DFA.
				idx := n*uint64(cs.id) + uint64(next.id)
				link := &Elr0CausalLink{
					pred: item,
				}
				if olditem, has := ns.itemsByState[idx]; !has {
					nitem := &Elr0ParseItem{
						dfaState: next,
						parentState: item.parentState,
						links: []*Elr0CausalLink{link},
						particle: lexerToken,
						parseStateId: ns.id,
					}
					ns.items = append(ns.items, nitem)
					ns.itemsByState[idx] = nitem
					fmt.Printf("SCAN (%d) %d %d pred{(%d) %d,%d}\n", ns.id, nitem.dfaState.id, nitem.parentState.id, link.pred.parseStateId, link.pred.dfaState.id, link.pred.parentState.id)
					// COMPLETE step: transition nondeterministically over epsilons.
					if enext, has := next.transitions[e]; has {
						idx := n*uint64(ns.id) + uint64(enext.id)
						if _, has := ns.itemsByState[idx]; !has {
							nitem := &Elr0ParseItem{
								dfaState: enext,
								parentState: ns,
								particle: nitem.particle,
								parseStateId: ns.id,
							}
							ns.items = append(ns.items, nitem)
							ns.itemsByState[idx] = nitem
							fmt.Printf("COMPLETE (%d) %d %d\n", ns.id, nitem.dfaState.id, nitem.parentState.id)
						}
					}
				} else {
					olditem.links = append(olditem.links, link)
				}
			}
			
			// If the parent state is cs, skip predictions step - cs is not 
			// complete yet and any possible reductions must be referenced by a
			// later state.
			if item.parentState == cs {
				continue
			}
			
			for nt, _ := range item.dfaState.reductions {
				for _, pitem := range item.parentState.items {
					// PREDICT step: transition on pendng nonterminal parent 
					// DFA edges if reductions might be possible from this state.
					if next, has := pitem.dfaState.transitions[nt]; has {
						idx := n*uint64(pitem.parentState.id) + uint64(next.id)
						link := &Elr0CausalLink{
							pred: pitem,
							cause: item,
						}
						if olditem, has := cs.itemsByState[idx]; !has {
							nitem := &Elr0ParseItem{
								dfaState: next,
								parentState: pitem.parentState,
								links: []*Elr0CausalLink{link},
								particle: nt,
								parseStateId: cs.id,
							}
							cs.items = append(cs.items, nitem)
							cs.itemsByState[idx] = nitem
							fmt.Printf("PREDICT (%d) %d %d pred={(%d) %d %d} cause={(%d) %d %d}\n", cs.id, nitem.dfaState.id, nitem.parentState.id, link.pred.parseStateId, link.pred.dfaState.id, link.pred.parentState.id, link.cause.parseStateId, link.cause.dfaState.id, link.cause.parentState.id)
							fmt.Printf("        nt=%s\n", nitem.particle.String())
							// COMPLETE step, again: transition nondeterministically over epsilons.
							if enext, has := next.transitions[e]; has {
								idx := n*uint64(cs.id) + uint64(enext.id)
								if _, has := cs.itemsByState[idx]; !has {
									nitem := &Elr0ParseItem{
										dfaState: enext,
										parentState: cs,
										particle: nitem.particle,
										parseStateId: cs.id,
									}
									cs.items = append(cs.items, nitem)
									cs.itemsByState[idx] = nitem
									fmt.Printf("COMPLETE (%d) %d %d\n", cs.id, nitem.dfaState.id, nitem.parentState.id)
								}
							}
						} else {
							olditem.links = append(olditem.links, link)
						}
					}
				}
			}
		}
		if len(ns.items) == 0 {
			return nil, p.ParseFailureError(cs)
		}
	}
	
	fmt.Println("end of parse, success")
	
	// Find the asterisk reduction in the final state.  If it's not there,
	// fail the parse - but this shouldn't happen and is probably a bug.
	var finalItem *Elr0ParseItem
	for _, item := range ns.items {
		if _, has := item.dfaState.reductions[p.grammar.Asterisk()]; has {
			finalItem = item
			fmt.Printf("FINAL ITEM: (%d) %d %d\n",ns.id,item.dfaState.id,item.parentState.id)
		}
	}
	if finalItem == nil {
		return nil, p.Error("no `* reduction in final state after successful parse - this is probably a parser bug")
	}
	
	// Now we must construct the AST from the parser states. 
	// First, walk down the link tree, flattening it into a binary tree with
	// only items used in the actual comprehension - we will also disambiguate
	// in this step.
	initial := &astEntry{
		node: &astNode{
			part: p.grammar.Asterisk(),
			rule: finalItem.dfaState.reductions[p.grammar.Asterisk()][0],
			first: 0,
			last: -1,
		},
		item: finalItem,
	}
	stack := []*astEntry{initial}
	for len(stack) > 0 {
		cs := stack[len(stack)-1]
		stack = stack[0:len(stack)-1]
		//fmt.Printf("    # (%d) %d %d\n", cs.item.parseStateId, cs.item.dfaState.id, cs.item.parentState.id)
 		switch len(cs.item.links) {
			case 0:
			case 1: {
				// XXX - need to do first/last updates here.
				leftEntry := &astEntry{node:&astNode{}}
				leftEntry.item = cs.item.links[0].pred
				leftEntry.node.parent = cs.node
				leftEntry.node.part = leftEntry.item.particle
				leftEntry.node.first = leftEntry.item.parseStateId
				cs.node.left = leftEntry.node
				//cs.node.part = leftEntry.item.particle
				if cs.item.links[0].cause != nil {
					rightEntry := &astEntry{node:&astNode{}}
					rightEntry.item = cs.item.links[0].cause
					rightEntry.node.parent = cs.node
					rightEntry.node.part = rightEntry.item.particle
					rightEntry.node.first = rightEntry.item.parseStateId
					cs.node.right = rightEntry.node
					reductions := rightEntry.item.dfaState.reductions[cs.node.part]
					if len(reductions) > 1 {
						panic("ambiguity resolution unimplemented")
					}
					if len(reductions) == 0 {
						panic(fmt.Sprintf("Right child with no reductions (nt=s) (%d) %d %d\n", /*cs.node.part.String(), */ rightEntry.item.parseStateId, rightEntry.item.dfaState.id, rightEntry.item.parentState.id))
					}
					cs.node.rule = reductions[0]
					cs.node.part = &syntaxValueNonterminal{
						GenericNonterminal: *cs.node.part.(*parser.GenericNonterminal),
						source: p,
					}
					stack = append(stack,rightEntry)
				} else {
					if vt, isValueTerm := cs.node.part.(*parser.ValueTerminal); isValueTerm {
						cs.node.part = &syntaxValueTerminal{
							ValueTerminal: *vt,
							source: p,
						}
					}
					cs.node.part = &syntaxValueTerminal{
						ValueTerminal: *parser.NewValueTerminal(cs.node.part,cs.node.part),
						source: p,
					}
				}
				 
				stack = append(stack, leftEntry)
			}
			default: {
				panic("ambiguity resolution unimplemented")
			}
		}
		if cs.node.rule == nil {
			cs.node.last = cs.node.first
		} else {
			cs.node.last = cs.node.first
			cs.node.first = 0
		}
	}
	if initial.node.right != nil {
		return initial.node.right, nil
	}
	return initial.node.left, nil
}

type astEntry struct {
	node *astNode
	item *Elr0ParseItem
}

type astNode struct {
	part parser.GrammarParticle
	rule parser.Production
	first, last int
	parent, left, right *astNode
}

func (an *astNode) Part() parser.GrammarParticle {
	return an.part
}

func (an *astNode) First() int {
	//nlist := []*astNode{an} 
	cn := an
	if cn.first == 0 {
		cn.first = cn.left.last + 1
	}
	return an.first
}

func (an *astNode) Last() int {
	return an.last
}

/*
type SyntaxValue interface {
	Supports(p Production) bool
	Value(p Production, idx int) SyntaxValue
}

type SyntaxValueFactory func(p Production, values []SyntaxValue) SyntaxValue
*/

type syntaxValueNonterminal struct {
	// iface parser.SyntaxValue
	parser.GenericNonterminal
	source parser.Parser
	value interface{}
}

type syntaxValueTerminal struct {
	parser.ValueTerminal
	source parser.Parser
}

func (svn *syntaxValueNonterminal) Value() interface{} {
	return svn.value
}

type valstackEntry struct {
	expanded bool
	node *astNode
}

func (an *astNode) Value() interface{} {
	if an.part.Terminal() {
		if vt, isValue := an.part.(*parser.ValueTerminal); isValue {
			return vt.Value()
		}
		return an.part
	}
	if vn, isValue := an.part.(*syntaxValueNonterminal); isValue {
		val := vn.Value()
		if val == nil {
			valstack := []*valstackEntry{&valstackEntry{node:an}}
			for len(valstack) > 0 {
				cvs := valstack[len(valstack)-1]
				if !cvs.expanded {
					for i := 0; i < cvs.node.NumChildren(); i++ {
						cn := cvs.node.Child(i).(*astNode)
						if cn.Part().Terminal() {
							continue
						}
						cvn, isValue := cn.Part().(*syntaxValueNonterminal)
						if isValue && cvn.Value() != nil {
							continue
						}
						valstack = append(valstack, &valstackEntry{node:cn})
					}
					cvs.expanded = true
					continue
				}
				valstack = valstack[0:len(valstack)-1]
				vals := make([]parser.SyntaxValue,cvs.node.NumChildren())
				for i := 0; i < len(vals); i++ {
					cnsv := cvs.node.Child(i).(*astNode).part.(parser.SyntaxValue)
					vals[i] = cnsv
				}		
				factory := cvs.node.Part().(*syntaxValueNonterminal).source.(*Elr0Parser).factory
				sval := factory(cvs.node.Rule(), vals)
				cvs.node.part.(*syntaxValueNonterminal).value = sval
			}
			return vn.Value()
		}
		return val
	}
	return an.part
}

func (an *astNode) Rule() parser.Production {
	return an.rule
}

func (an *astNode) NumChildren() int {
	if an.rule != nil {
		return an.rule.RhsLen()
	}
	return 0
}

func NodeType(an *astNode) string {
	if an.left != nil && an.right != nil {
		return "REDUCTION"
	}
	if an.left != nil {
		return "SCAN"
	}
	return "LEAF"
}

func DumpTree(an *astNode) {
	DumpTreeR(an, make(map[uintptr]int))
}

func DumpTreeR(an *astNode, canon map[uintptr]int) {
	fmt.Println(nodeStr(an, canon))
	if an.left != nil {
		DumpTreeR(an.left, canon)
	}
	if an.right != nil {
		DumpTreeR(an.right, canon)
	}
}

func nodeStr(an *astNode, canon map[uintptr]int) string {
	var id, lid, rid int
	var has bool
	ptr := reflect.ValueOf(an).Pointer()
	if id, has = canon[ptr]; !has {
		id = len(canon) 
		canon[ptr] = id
	}
	var leftstr, rightstr string
	if an.left == nil {
		leftstr = "-"
	} else {
		lptr := reflect.ValueOf(an.left).Pointer()
		if lid, has = canon[lptr]; !has {
			lid = len(canon)
			canon[lptr] = lid
		}
		leftstr = fmt.Sprintf("[%d]", lid)
	}
	if an.right == nil {
		rightstr = "-"
	} else {
		rptr := reflect.ValueOf(an.right).Pointer()
		if rid, has = canon[rptr]; !has {
			rid = len(canon)
			canon[rptr] = rid
		}
		rightstr = fmt.Sprintf("[%d]", rid)
	}
	var rulestr, partstr string
	if an.rule == nil {
		rulestr = "{}"
	} else {
		rulestr = fmt.Sprintf("{%s}", an.rule.String())
	}
	if an.part == nil {
		partstr = "?"
	} else {
		partstr = an.part.String()
	}
	var valstr = "-"
	if vt, isvt := an.part.(*parser.ValueTerminal); isvt {
		val := vt.Value()
		if vts, isstr := val.(string); isstr {
			valstr = fmt.Sprintf("\"%s\"", vts) 
		} else if vts, isstr := val.(Stringable); isstr {
			valstr = fmt.Sprintf("\"%s\"", vts.String())
		} else {
			valstr = fmt.Sprintf("*0x%8.8X", reflect.ValueOf(val).Pointer())
		}
	}
	return fmt.Sprintf("[%d: %s %d %d %s %s <%s, %s> %s]\n", id, NodeType(an), an.first, an.last, partstr, rulestr, leftstr, rightstr, valstr)
}

func (an *astNode) Child(idx int) parser.SyntaxTreeNode {
	cn := an.right
	for i := 0; i < an.rule.RhsLen()-idx-1; i++ {
		if cn == nil {
			return nil 
		}
		cn = cn.left
	}
	return cn
}
