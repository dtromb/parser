package parser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
)

type lr0Item struct {
	rule     ProductionRule
	caretPos int
	hc       uint32
}

type sortedLR0ItemSet []*lr0Item

type lr0ItemSet struct {
	items  sortedLR0ItemSet
	sorted bool
	hc     uint32
}

type earleyItemState struct {
	id          int
	kernel      bool
	itemSet     *lr0ItemSet
	transitions map[Term]*earleyItemState
	reductions  []ProductionRule
	hc          uint32
}

type earleyParserGenerator struct {
	grammar   IndexedGrammar
	prodIndex ProductionGrammarIndex
	nullIndex NullabilityGrammarIndex
	states    []*earleyItemState
}

type earleyParserDfaState struct {
	transitions    map[int]int
	reductions     []ProductionRule
	reductionIndex map[int][]ProductionRule
}

type earleyParser struct {
	grammar          Grammar
	generator        *earleyParserGenerator
	dfa              []earleyParserDfaState
	acceptStateIndex int
	epsNt            map[int]Term
}

type earleyParserStateLink struct {
	pred  *earleyParserEntry
	cause *earleyParserEntry
}

type earleyParserEntry struct {
	dfaStateId  uint32
	parentIndex uint32
	links       []*earleyParserStateLink
	linkIndex   map[uint64][]int
	token       Token
}

type earleyParserEntryList struct {
	entries []*earleyParserEntry
	index   map[uint64]int
}
type earlyParserState struct {
	state  []*earleyParserEntryList
	lexer  LexerState
	parser *earleyParser
}

func (lr *lr0Item) String() string {
	var buf []byte
	pr := lr.rule
	buf = []byte(fmt.Sprintf("%d: %s", pr.Id(), TermToString(pr.Lhs())))
	buf = append(buf, " := "...)
	for i := 0; i < pr.RhsLen(); i++ {
		if lr.caretPos == i {
			buf = append(buf, ". "...)
		}
		term := pr.Rhs(i)
		//fmt.Printf("%d: %s\n", i, TermToString(term))
		buf = append(buf, TermToString(term)...)
		buf = append(buf, byte(' '))
	}
	buf = buf[0 : len(buf)-1]
	if lr.caretPos == pr.RhsLen() {
		buf = append(buf, " ."...)
	}
	return string(buf)
}

func (lr *lr0Item) HashCode() uint32 {
	if lr.hc == 0 {
		lr.hc = lr.rule.HashCode()
		lr.hc = (lr.hc >> 5) | (lr.hc << 27)
		lr.hc ^= uint32(lr.caretPos)
	}
	return lr.hc
}

func (lr *lr0Item) Equals(a interface{}) bool {
	if lra, ok := a.(*lr0Item); ok {
		return lr.CompareTo(lra) == 0
	}
	if lra, ok := a.(lr0Item); ok {
		return lr.CompareTo(&lra) == 0
	}
	panic("TYPE: " + reflect.TypeOf(a).String())
	return false
}

func (lr *lr0Item) CompareTo(a *lr0Item) int {
	if lr.rule.Id() < a.rule.Id() {
		return -1
	}
	if lr.rule.Id() > a.rule.Id() {
		return 1
	}
	if lr.caretPos < a.caretPos {
		return -1
	}
	if lr.caretPos > a.caretPos {
		return 1
	}
	return 0
}

func (sis sortedLR0ItemSet) Len() int           { return len(sis) }
func (sis sortedLR0ItemSet) Less(i, j int) bool { return (sis[i]).CompareTo(sis[j]) < 0 }
func (sis sortedLR0ItemSet) Swap(i, j int)      { sis[i], sis[j] = sis[j], sis[i] }

func (is *earleyItemState) writeStateInfo(out io.Writer) {
	kmark := "k"
	if !is.kernel {
		kmark = "nk"
	}
	out.Write([]byte(fmt.Sprintf("[%d] %s\n", is.id, kmark)))
	for _, lr := range is.itemSet.items {
		out.Write([]byte(fmt.Sprintf("   %s\n", lr.String())))
	}
	for t, ns := range is.transitions {
		out.Write([]byte(fmt.Sprintf("   %s => [%d]\n", TermToString(t), ns.id)))
	}
	for _, pr := range is.reductions {
		out.Write([]byte(fmt.Sprintf("   <- %s\n", ProductionRuleToString(pr))))
	}
}

func (is *earleyItemState) HashCode() uint32 {
	return is.itemSet.HashCode()
}

func (is *earleyItemState) Equals(o interface{}) bool {
	if ois, ok := o.(*earleyItemState); ok {
		return is.itemSet.Equals(ois.itemSet)
	}
	return false
}

func (is *lr0ItemSet) HashCode() uint32 {
	if is.hc == 0 {
		for _, k := range is.items {
			is.hc ^= k.HashCode()
		}
	}
	return is.hc
}

func (is *lr0ItemSet) Equals(o interface{}) bool {
	if !is.sorted {
		sort.Sort(is.items)
		is.sorted = true
	}
	if iso, ok := o.(*lr0ItemSet); ok {
		if !iso.sorted {
			sort.Sort(iso.items)
			iso.sorted = true
		}
		if len(is.items) != len(iso.items) {
			return false
		}
		for i, x := range is.items {
			if !iso.items[i].Equals(x) {
				fmt.Printf("%s != %s\n", iso.items[i].String(), x.String())
				return false
			}
		}
		return true
	}
	return false
}

func (ep *earleyParserGenerator) isInitialForm(lr *lr0Item) bool {
	if lr.caretPos != 0 {
		return false
	}
	if lr.rule.Lhs().Id() != ep.grammar.Asterisk().Id() {
		return false
	}
	if lr.rule.RhsLen() != 2 {
		return false
	}
	if lr.rule.Rhs(0).Terminal() || lr.rule.Rhs(0).Special() {
		return false
	}
	return lr.rule.Rhs(1).Id() == ep.grammar.Bottom().Id()
}

func (ep *earleyParserGenerator) ItemSetClosure(seed *lr0ItemSet) (*earleyItemState, *earleyItemState, error) {

	fmt.Println("=======\nCLOSURE START")
	expandedNt := NewHashSet()
	kStack := []*lr0Item{}
	nkStack := []*lr0Item{}
	for _, item := range seed.items {
		if item.caretPos == 0 && !ep.isInitialForm(item) {
			fmt.Println("NK: " + item.String())
			nkStack = append(nkStack, item)
		} else {
			fmt.Println("K: " + item.String())
			kStack = append(kStack, item)
		}
	}
	kSet := NewHashSet()  // of *lr0Item
	nkSet := NewHashSet() // of *lr0Item
	for len(nkStack) > 0 || len(kStack) > 0 {
		fmt.Printf("   (loop klen = %d, nklen=%d)\n", len(kStack), len(nkStack))
		//var kItem bool
		var citem *lr0Item = nil
		if len(kStack) > 0 {
			//kItem = true
			if len(kStack) > 0 {
				fmt.Printf("   (kStack[0] == %s)\n", kStack[0])
			}
			if len(kStack) > 1 {
				fmt.Printf("   (kStack[1] == %s)\n", kStack[1])
			}
			citem = kStack[len(kStack)-1]
			kStack = kStack[0 : len(kStack)-1]
			kSet.Add(citem)
		} else {
			//kItem = false
			citem = nkStack[len(nkStack)-1]
			nkStack = nkStack[0 : len(nkStack)-1]
			nkSet.Add(citem)
		}
		fmt.Println("CITEM: " + citem.String())
		fmt.Printf("   (new klen = %d, nklen=%d)\n", len(kStack), len(nkStack))
		if citem.caretPos < citem.rule.RhsLen() {
			nextTerm := citem.rule.Rhs(citem.caretPos)
			fmt.Println("Next term is " + TermToString(nextTerm))
			if !nextTerm.Terminal() {
				if _, has := expandedNt.Has(nextTerm); has {
					fmt.Println("skipping already expanded term")
					continue
				}
				expandedNt.Add(nextTerm)
				fmt.Printf("%d productions for %s\n", len(ep.prodIndex.GetProductions(nextTerm)), TermToString(nextTerm))
				for _, np := range ep.prodIndex.GetProductions(nextTerm) {

					newItem := &lr0Item{rule: np}
					if _, has := nkSet.Has(newItem); has {
						continue
					}
					fmt.Println("Adding item " + newItem.String())
					nkStack = append(nkStack, newItem)
				}
			}
			if ep.nullIndex.IsNullable(nextTerm) {
				newItem := &lr0Item{rule: citem.rule, caretPos: citem.caretPos + 1}
				if _, has := nkSet.Has(newItem); has {
					continue
				}
				nkStack = append(nkStack, newItem)
			}
		}
	}
	it := kSet.OpenCursor()
	kState := &earleyItemState{
		id:          len(ep.states),
		kernel:      true,
		itemSet:     &lr0ItemSet{items: make([]*lr0Item, 0, len(kStack))},
		transitions: make(map[Term]*earleyItemState),
	}
	for it.HasMore() {
		kState.itemSet.items = append(kState.itemSet.items, it.Next().(*lr0Item))
	}
	sort.Sort(kState.itemSet.items)
	//kState.writeStateInfo(os.Stdout)
	if len(kState.itemSet.items) == 0 {
		panic("empty kernel state")
	}
	//ep.states = append(ep.states, kState)
	//fmt.Println()
	nkState := &earleyItemState{
		id:          len(ep.states) + 1,
		kernel:      false,
		itemSet:     &lr0ItemSet{items: make([]*lr0Item, 0, nkSet.Size())},
		transitions: make(map[Term]*earleyItemState),
	}
	it = nkSet.OpenCursor()
	for it.HasMore() {
		st := it.Next().(*lr0Item)
		//fmt.Println("wtf: " + st.String())
		nkState.itemSet.items = append(nkState.itemSet.items, st)
	}
	sort.Sort(nkState.itemSet.items)
	if len(nkState.itemSet.items) > 0 {
		//ep.states = append(ep.states, nkState)
		kState.transitions[ep.grammar.Epsilon()] = nkState
		//nkState.writeStateInfo(os.Stdout)
	} else {
		nkState = nil
	}
	//fmt.Println()
	return nkState, kState, nil
}

//func (is lr0ItemSet) Closure() lr0ItemSet {
//
//}

func GenerateEarleyParser(g Grammar) (Parser, error) {
	ig := GetIndexedGrammar(g)
	parserGen := &earleyParserGenerator{grammar: ig}
	idxIf, err := ig.GetIndex(GrammarIndexTypeProduction)
	if err != nil {
		return nil, err
	}
	parserGen.prodIndex = idxIf.(ProductionGrammarIndex)

	idxIf, err = ig.GetIndex(GrammarIndexTypeNullability)
	if err != nil {
		return nil, err
	}
	fmt.Println(reflect.TypeOf(idxIf).String())
	parserGen.nullIndex = idxIf.(NullabilityGrammarIndex)
	initialItem := &lr0Item{
		rule: parserGen.prodIndex.GetInitialProduction(),
	}
	initialItem = initialItem
	initialItemSet := &lr0ItemSet{
		items: sortedLR0ItemSet([]*lr0Item{initialItem}),
	}
	nk, k, err := parserGen.ItemSetClosure(initialItemSet)
	if err != nil {
		return nil, err
	}
	allStates := NewHashSet()
	stateStack := []*earleyItemState{nk, k}
	allStates.Add(nk, k)
	parserGen.states = append(parserGen.states, k)
	parserGen.states = append(parserGen.states, nk)
	for len(stateStack) > 0 {
		cs := stateStack[len(stateStack)-1]
		transitionSeeds := make(map[Term]Hashset)
		stateStack = stateStack[0 : len(stateStack)-1]
		for _, item := range cs.itemSet.items {
			if item.caretPos == item.rule.RhsLen() {
				cs.reductions = append(cs.reductions, item.rule)
			} else {
				nextTerm := item.rule.Rhs(item.caretPos)
				if _, has := transitionSeeds[nextTerm]; !has {
					transitionSeeds[nextTerm] = NewHashSet()
				}
				transitionSeeds[nextTerm].Add(&lr0Item{rule: item.rule, caretPos: item.caretPos + 1})
			}
		}
		for term, seed := range transitionSeeds {
			if term.Id() == parserGen.grammar.Epsilon().Id() {
				continue
			}
			set := &lr0ItemSet{items: make([]*lr0Item, 0, seed.Size())}
			it := seed.OpenCursor()
			for it.HasMore() {
				set.items = append(set.items, it.Next().(*lr0Item))
			}
			sort.Sort(set.items)
			//fmt.Printf("Seed set [%d] %s:\n", cs.id, TermToString(term))
			//for _, k := range set.items {
			//	fmt.Println("   " + k.String())
			//}
			trNkState, trKState, err := parserGen.ItemSetClosure(set)
			if err != nil {
				return nil, err
			}
			if cState, has := allStates.Has(trKState); has {
				trKState = cState.(*earleyItemState)
			} else {
				trKState.id = len(parserGen.states)
				parserGen.states = append(parserGen.states, trKState)
				allStates.Add(trKState)
				stateStack = append(stateStack, trKState)
				//fmt.Println("NEW STATE")
				//trKState.writeStateInfo(os.Stdout)
				//fmt.Println()
			}
			cs.transitions[term] = trKState
			if trNkState != nil {
				if cState, has := allStates.Has(trNkState); has {
					trNkState = cState.(*earleyItemState)
				} else {
					trNkState.id = len(parserGen.states)
					parserGen.states = append(parserGen.states, trNkState)
					allStates.Add(trNkState)
					stateStack = append(stateStack, trNkState)
					//fmt.Println("NEW STATE")
					//trNkState.writeStateInfo(os.Stdout)
					//fmt.Println()
				}
				trKState.transitions[parserGen.grammar.Epsilon()] = trNkState
			} else {
				fmt.Printf("nk state of %d empty\n", trKState.id)
			}
		}
	}
	for i, st := range parserGen.states {
		fmt.Printf("State #%d:\n", i)
		st.writeStateInfo(os.Stdout)
		fmt.Println("-----\n")
	}
	parser := &earleyParser{
		grammar:   g,
		generator: parserGen,
		dfa:       make([]earleyParserDfaState, len(parserGen.states)),
	}
	for i := 0; i < len(parser.dfa); i++ {
		parser.dfa[i].transitions = make(map[int]int)
		for t, ns := range parserGen.states[i].transitions {
			parser.dfa[i].transitions[int(t.Id())] = ns.id
		}
		parser.dfa[i].reductions = make([]ProductionRule, len(parserGen.states[i].reductions))
		parser.dfa[i].reductionIndex = make(map[int][]ProductionRule)
		for j := 0; j < len(parserGen.states[i].reductions); j++ {
			pr := parserGen.states[i].reductions[j]
			parser.dfa[i].reductions[j] = pr
			if pr.Lhs().Id() == parser.grammar.Asterisk().Id() {
				fmt.Printf("ACCEPT STATE IS %d\n", i)
				parser.acceptStateIndex = i
			}
			if _, has := parser.dfa[i].reductionIndex[int(pr.Lhs().Id())]; !has {
				parser.dfa[i].reductionIndex[int(pr.Lhs().Id())] = []ProductionRule{pr}
			} else {
				parser.dfa[i].reductionIndex[int(pr.Lhs().Id())] = append(parser.dfa[i].reductionIndex[int(pr.Lhs().Id())], pr)
			}
		}
	}
	// Retain the null element index if there are nullables.
	parser.epsNt = make(map[int]Term)
	if parserGen.nullIndex.HasNullableNt() {
		for _, epsNt := range parserGen.nullIndex.GetNullableNonterminals() {
			parser.epsNt[int(epsNt.Id())] = epsNt
		}
	}
	return parser, nil
}

func (p *earleyParser) Grammar() Grammar {
	return p.grammar
}

func (p *earleyParser) Open(lexState LexerState) (ParserState, error) {
	ps := &earlyParserState{
		parser: p,
		lexer:  lexState,
	}
	return ps, nil
}

func (ps *earlyParserState) Parser() Parser {
	return ps.parser
}

func (ps *earlyParserState) LexerState() LexerState {
	return ps.lexer
}

func (ps *earlyParserState) Parse() (ParseTreeNode, error) {
	// Create the state array and inital state.
	ps.state = make([]*earleyParserEntryList, 1, 64)
	ps.state[0] = &earleyParserEntryList{
		entries: make([]*earleyParserEntry, 1, 2),
		index:   make(map[uint64]int),
	}
	ps.state[0].entries[0] = &earleyParserEntry{}
	ps.state[0].index[0] = 0
	i := 0
	eps := ps.parser.grammar.Epsilon().Id()

	// First eps transition must be done manually.
	if nk, has := ps.parser.dfa[0].transitions[int(eps)]; has {
		fmt.Printf("Will add (%d,0) to [0]\n", nk)
		ns := &earleyParserEntry{
			dfaStateId: uint32(nk),
			links:      []*earleyParserStateLink{},
			linkIndex:  make(map[uint64][]int),
		}
		ps.state[0].index[uint64(nk)<<32] = 1
		ps.state[0].entries = append(ps.state[0].entries, ns)
	}

	canAccept := false
	for {
		hasMore, err := ps.lexer.HasMoreTokens()
		if err != nil {
			return nil, err
		}
		if hasMore {
			fmt.Printf("HAS MORE (%d)\n", i)
			canAccept = false
			// Get the next token.
			nextTok, err := ps.lexer.NextToken()
			if err != nil {
				return nil, err
			}
			fmt.Printf("Token %d is: %s(%s)\n", i+1, TermToString(nextTok.Terminal()), nextTok.Literal())
			// Setup S_{x+1}
			nsl := &earleyParserEntryList{
				entries: []*earleyParserEntry{},
				index:   make(map[uint64]int),
			}
			ps.state = append(ps.state, nsl)
			canContinue := false
			/** foreach item in S_i: */
			cs := ps.state[i]
			for j := 0; j < len(cs.entries); j++ {
				fmt.Printf(" Considering [%d]:%d\n", i, j)
				item := cs.entries[j]
				/** (state,parent) <- item */
				state := &ps.parser.dfa[item.dfaStateId]
				parentId := item.parentIndex
				parent := ps.state[parentId]
				/** k <- goto(state, x{i+1}) */
				k, has := state.transitions[int(nextTok.Terminal().Id())]
				/** if k != nil: */
				if has {
					canContinue = true
					fmt.Printf("  goto(%d,%s) = %d\n", item.dfaStateId, TermToString(nextTok.Terminal()), k)
					/** add(k,parent) to S_{i+1} */
					var ns *earleyParserEntry
					key := (uint64(k) << 32) | uint64(parentId)
					if ent, has := nsl.index[key]; has {
						ns = nsl.entries[ent]
					} else {
						fmt.Printf("  Will add (%d,%d) to [%d]\n", k, parentId, i+1)
						ns = &earleyParserEntry{
							dfaStateId:  uint32(k),
							parentIndex: parentId,
							links:       make([]*earleyParserStateLink, 0, 4),
							linkIndex:   make(map[uint64][]int),
							token:       nextTok,
						}
						nsl.index[key] = len(nsl.entries)
						nsl.entries = append(nsl.entries, ns)
					}
					/** add link(&item, nil) to (k, parent) in S_{i+1} */
					fmt.Printf("  Will link <[%d]:%d(%d,%d),nil> from [%d]:%d(%d,%d)\n", i, j, item.dfaStateId, item.parentIndex, i+1, len(nsl.entries)-1, k, parentId)
					link := &earleyParserStateLink{
						pred: item,
					}
					if _, has := ns.linkIndex[0]; has {
						ns.linkIndex[0] = append(ns.linkIndex[0], len(ns.links))
					} else {
						ns.linkIndex[0] = []int{len(ns.links)}
					}
					ns.links = append(ns.links, link)
					/** nk <- goto(k,`e) */
					nk, has := ps.parser.dfa[k].transitions[int(eps)]
					/** if nk != nil: */
					if has {
						fmt.Printf("  goto(%d,`e) = %d\n", k, nk)
						/** add(nk,i+1) to S_{i+1} */
						key = (uint64(nk) << 32) | uint64(i+1)
						if ent, has := nsl.index[key]; has {
							ns = nsl.entries[ent]
						} else {
							fmt.Printf("  Will add (%d,%d) to [%d]\n", nk, i+1, i+1)
							ns = &earleyParserEntry{
								dfaStateId:  uint32(nk),
								parentIndex: uint32(i + 1),
								links:       []*earleyParserStateLink{},
								linkIndex:   make(map[uint64][]int),
							}
							nsl.index[key] = len(nsl.entries)
							nsl.entries = append(nsl.entries, ns)
						}
					}
				}
				/** if parent == i: continue */
				if parentId == uint32(i) {
					continue
				}
				/** foreach A->a in completed(state): */
				for _, pr := range state.reductions {
					fmt.Println(" Can reduce via " + ProductionRuleToString(pr))
					a := pr.Lhs().Id()
					/** foreach pitem in S_{parent}: */
					for k := 0; k < len(parent.entries); k++ {
						pitem := parent.entries[k]
						fmt.Printf(" Reduce considering [%d]:%d(%d,%d)\n", parentId, k, pitem.dfaStateId, pitem.parentIndex)
						// record accept input condition
						if parentId == 0 && k == 0 {
							fmt.Println("ACCEPT OK")
							canAccept = true
						}
						/** (pstate,pparent) <- pitem */
						pstate := &ps.parser.dfa[pitem.dfaStateId]
						pparentId := pitem.parentIndex
						// pparent := ps.state[pparentId]
						/** k <- goto(pstate,A) */
						k, has := pstate.transitions[int(a)]
						/** if k != nil: */
						var ns *earleyParserEntry
						if has {
							fmt.Printf("  goto(%d,%s) = %d\n", pitem.dfaStateId, TermToString(pr.Lhs()), k)
							/** add (k,pparent) to S_i */
							key := (uint64(k) << 32) | uint64(pparentId)
							if ent, has := cs.index[key]; has {
								ns = cs.entries[ent]
							} else {
								fmt.Printf("  Will add (%d,%d) to [%d]\n", k, pparentId, i)
								ns = &earleyParserEntry{
									dfaStateId:  uint32(k),
									parentIndex: uint32(pparentId),
									links:       make([]*earleyParserStateLink, 0, 4),
									linkIndex:   make(map[uint64][]int),
								}
								cs.index[key] = len(cs.entries)
								cs.entries = append(cs.entries, ns)
							}
							/** add link (&pitem,&item) to (k,pparent) in S_i */
							fmt.Printf("  Will link <[%d]:%d(%d,%d),[%d]:%d(%d,%d)> from [%d]:%d(%d,%d)\n", parentId, k, pitem.dfaStateId, pitem.parentIndex, i+1,
								i, j, item.dfaStateId, item.parentIndex,
								len(cs.entries)-1, k, pparentId)
							link := &earleyParserStateLink{
								pred:  pitem,
								cause: item,
							}
							linkKey := (uint64(item.dfaStateId) << 32) | uint64(item.parentIndex)
							if _, has := ns.linkIndex[linkKey]; has {
								ns.linkIndex[linkKey] = append(ns.linkIndex[linkKey], len(ns.links))
							} else {
								ns.linkIndex[linkKey] = []int{len(ns.links)}
							}
							ns.links = append(ns.links, link)
							/** nk <- goto(k,`e) */
							nk, has := ps.parser.dfa[k].transitions[int(eps)]
							/** if nk != nil: */
							if has {
								fmt.Printf("  goto(%d,`e) = %d\n", k, nk)
								/** add (nk,i) to S_i */
								key := (uint64(nk) << 32) | uint64(i)
								if ent, has := cs.index[key]; has {
									ns = cs.entries[ent]
								} else {
									fmt.Printf("  Will add (%d,%d) to [%d]\n", nk, i, i)
									ns = &earleyParserEntry{
										dfaStateId:  uint32(nk),
										parentIndex: uint32(i),
										links:       []*earleyParserStateLink{},
									}
									cs.index[key] = len(cs.entries)
									cs.entries = append(cs.entries, ns)
								}
							}
						}
					}
				}
			}
			if !canContinue {
				return nil, errors.New(fmt.Sprintf("parse error at token %d\n", i+1))
			}
			i++
		} else {
			if canAccept {
				break
			}
			return nil, errors.New("unexpected end of input")
		}
	}

	for idx, el := range ps.state {
		fmt.Printf("<%d>\n", idx)
		for _, ent := range el.entries {
			fmt.Printf("  (%d,%d)\n", ent.dfaStateId, ent.parentIndex)
			for _, ln := range ent.links {
				if ln.cause != nil {
					fmt.Printf("      -><(%d,%d),(%d,%d)>\n", ln.pred.dfaStateId, ln.pred.parentIndex, ln.cause.dfaStateId, ln.cause.parentIndex)
				} else {
					fmt.Printf("      -><(%d,%d),nil>\n", ln.pred.dfaStateId, ln.pred.parentIndex)
				}
			}
		}
	}

	// The parser accepted the input.   We must now construct the AST by
	// tracing derivations back through the entry list links.
	stack := make([]*earleyParseRecord, 1, 32)

	// First scan the final state to find the initial entry - this corresponds
	// to the final production rule reduction and therefore the bottom / first
	// element of the evaluation stack.
	var initialEntry *earleyParserEntry
	for _, entry := range ps.state[len(ps.state)-1].entries {
		if entry.dfaStateId == uint32(ps.parser.acceptStateIndex) {
			initialEntry = entry
			break
		}
	}
	if initialEntry == nil {
		return nil, errors.New("no initial entry found in final state after successful parse")
	}

	var ast *earleyParseTreeNode

	// Walk back manually through the initial production.
	initialNt := ps.parser.grammar.Asterisk()

	// Setup the stack bottom.
	stack[0] = &earleyParseRecord{
		//nt:    ps.parser.grammar.Asterisk(),
		nt:     initialNt,
		entry:  initialEntry,
		target: &ast,
		//pr:     ps.parser.dfa[ps.parser.acceptStateIndex].reductions[0],
	}

	for len(stack) > 0 {
		ce := stack[len(stack)-1]
		if ce.pr == nil {
			state := &ps.parser.dfa[ce.entry.dfaStateId]
			prs, has := state.reductionIndex[int(ce.nt.Id())]
			if !has {
				return nil, errors.New(fmt.Sprintf("missing reduction in state after successful parse (%d,%d)\n", ce.entry.dfaStateId, ce.entry.parentIndex))
			}
			if len(prs) > 1 {
				return nil, errors.New("R/R ambiguity encountered after successful parse, ambiguity resolution not yet implemented")
			}
			ce.pr = prs[0]
			ce.x = &earleyParseTreeNode{
				parser:   ps.parser,
				term:     ce.pr.Lhs(),
				rule:     ce.pr,
				children: make([]*earleyParseTreeNode, ce.pr.RhsLen()),
			}
			fmt.Printf("Select PR: %s\n", ProductionRuleToString(ce.pr))
		}
		if ce.n == ce.pr.RhsLen() {
			*ce.target = ce.x
			stack = stack[0 : len(stack)-1]
			continue
		}
		//sym := ce.pr.Rhs(ce.n)
		ce.n++
		sym := ce.pr.Rhs(ce.pr.RhsLen() - ce.n)
		fmt.Printf("SYM: %s\n", TermToString(sym))
		if sym.Terminal() {
			nx := &earleyParseTreeNode{
				parser: ps.parser,
				term:   sym,
				token:  ce.entry.token,
			}
			ce.x.children[ce.pr.RhsLen()-ce.n] = nx
			plinks, has := ce.entry.linkIndex[0]
			if !has {
				return nil, errors.New("missing predecessor link for terminal transition after successful parse")
			}
			ce.entry = ce.entry.links[plinks[0]].pred
		} else {
			if _, isEps := ps.parser.epsNt[int(sym.Id())]; isEps {
				return nil, errors.New("epsilon deriviation encountered after successful parse - not yet implemented")
			}
			var causeLink *earleyParserStateLink
			for _, link := range ce.entry.links {
				if link.cause == nil {
					continue
				}
				if causeLink != nil {
					return nil, errors.New("causal ambiguity encountered after successful parse - amgiuity resolution not yet implemented")
				}
				causeLink = link
			}
			if causeLink == nil {
				return nil, errors.New(fmt.Sprintf("missing causal link after successful parse (%d,%d)\n", ce.entry.dfaStateId, ce.entry.parentIndex))
			}
			fmt.Printf("rhslen: %d, children len: %d, index: %d\n", ce.pr.RhsLen(), len(ce.x.children), ce.pr.RhsLen()-ce.n)
			ne := &earleyParseRecord{
				nt:     sym,
				entry:  causeLink.cause,
				target: &ce.x.children[ce.pr.RhsLen()-ce.n],
			}
			ce.entry = causeLink.pred
			fmt.Printf("Push <%s,(%d,%d)>\n", TermToString(sym), causeLink.cause.dfaStateId, causeLink.cause.parentIndex)
			stack = append(stack, ne)
		}
	}
	return ast, nil
}

type earleyParseRecord struct {
	nt     Term
	entry  *earleyParserEntry
	pr     ProductionRule
	n      int
	x      *earleyParseTreeNode
	target **earleyParseTreeNode
}

type earleyParseTreeNode struct {
	parser   *earleyParser
	term     Term
	rule     ProductionRule
	children []*earleyParseTreeNode
	token    Token
}

func (pn *earleyParseTreeNode) Parser() Parser {
	return pn.parser
}

func (pn *earleyParseTreeNode) Term() Term {
	return pn.term
}

func (pn *earleyParseTreeNode) Token() Token {
	return pn.token
}

func (pn *earleyParseTreeNode) Production() ProductionRule {
	return pn.rule
}

func (pn *earleyParseTreeNode) NumChildren() int {
	return len(pn.children)
}

func (pn *earleyParseTreeNode) Child(idx int) ParseTreeNode {
	if idx < 0 || idx >= len(pn.children) {
		return nil
	}
	return pn.children[idx]
}

func (pn *earleyParseTreeNode) Children() []ParseTreeNode {
	ret := make([]ParseTreeNode, len(pn.children))
	for i, c := range pn.children {
		ret[i] = c
	}
	return ret
}
