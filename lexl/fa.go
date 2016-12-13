package lexl

import (
	"errors"
	"fmt"
	"sort"
)

// Ndfa represents an NDFA lexer construction.  It is comprised of a set of 
// unique NdfaState and its primary use is to construct the lexer DFA.   In
// an NDFA, state transitions are not necessarily unique, so direct users must
// either backtrace or maintain a state tree.
type Ndfa interface {
	// Returns the number of NdfaState in this Ndfa.
	NumStates() int
	
	// Returns the given state of the Ndfa, or nil if the index is out of range.
	State(idx int) NdfaState
	
	// Attempt to transform the NDFA to a DFA.  
	TransformToDfa() (Dfa, error)
}

// NdfaState represents a single state in a lexer NDFA construction.  Each
// state has a set of transitions which may either be character literals or 
// character ranges.  These may overlap and each point to a possible next state 
// if a valid input character matching the transition is consumed.  In addition,
// it may have epsilon transitions which do not consume any characters.
type NdfaState interface {
	
	// Returns the ID of this NdfaState
	ID() int
	
	// Returns a slice of literal characters this state may consume.
	Literals() []rune
	
	// Returns a slice of CharacterRange representing characters this state may consume.
	Ranges() []CharacterRange
	
	// Returns a slice of NdfaState the Ndfa may transition into from this state through
	// a transition consuming a character returned by Literals().
	LiteralTransitions(c rune) []NdfaState
	
	// Returns a slice of NdfaState the Ndfa may transition into from this state through
	// a transition consuming a character in a CharacterRange returned by Ranges().
	RangeTransitions(r CharacterRange) []NdfaState
	
	// Returns a slice of NdfaState the Ndfa may transition into from this state
	// without consuming any characters.
	EpsilonTransitions() []NdfaState
	
	// Queries the NdfaState, returning a set of NdfaState the Ndfa may transition into
	// from this state after consuming the given character (through a literal or range transition).
	Query(c rune) []NdfaState
	
	// CanAccept() bool UNIMPLEMENTED
	// AcceptTerms() []parser.Term UNIMPLEMENTED
}

// Dfa represents a DFA lexer construction.  It is comprised of a set of 
// unique DfaState and is intended for direct use by lexer implementations.   
// In a NDFA, state transitions are unique - the next state given a specific
// input is determined, so no backtracing is necessary (unless longest-match
// functionality or backreferences are required).
type Dfa interface {
	// Returns the number of DfaState in the DFA.
	NumStates() int
	
	// Returns the state with the given index, or nil if the index is out of range.
	State(idx int) DfaState
	
	// Returns the number of terminal names referenced by the DFA.
	NumTerminals() int
	
	// Returns the terminal name with the given index, or empty if the index is out of range.
	Terminal(idx int) string
}

// DfaState represents a single state in a lexer DFA construction.  Each
// state has a set of transitions which are represented as intervals.  The 
// intervals cover the codepoint space and some accept no inputs in their range.
// These may not overlap and each points to the next state reached if a valid 
// input character matching the interval is consumed.  The only actions possible
// are consuming a character matching an interval, or accepting a token for the
// lexer to return without matching a character.  In both cases the next state
// is determined.
type DfaState interface {
	
	// Returns the ID of this DfaState
	ID() int
	
	// Returns the number of target intervals this state defines.
	NumIntervals() int
	
	// Returns the least character matched by the interval with the given index.
	IntervalLower(idx int) rune
	
	// Returns the next state the Dfa will transition into if the interval with the 
	// given index is used to transition.
	IntervalTransition(idx int) DfaState
	
	// Returns true iff this state can transition by emitting a terminal instead of 
	// conduming a character.
	CanAccept() bool
	
	// Returns the dfa code of the terminal and the next DfaState reached by the Dfa
	// after an accept transition.
	AcceptTransition() (int, DfaState)
	
	// Query the Dfa, returning the next state the Dfa will reach if it transitions
	// after consuming the given character, or nil if no transition on this character
	// is possible.
	Query(c rune) DfaState
}

///


// Interface semantic representations of match items must implement to generate their states.
type ndfaStateGenerator interface {
	GenerateNdfaStates() ([]*stdNdfaState, error)
}

type stdDfa struct {
	dfa       []stdDfaState			// Slice of all states, state.id indexed
	terminals []string				// Slice of all terminal names, indexed by their dfa code
}

type stdNdfa struct {
	states []*stdNdfaState			// Slice of all states
}

type stdNdfaState struct {
	id                int			// State number
	termdefIndex	  int			// The linear lexical ID of the termdef that generated the state.
	accepting         bool			// True iff a terminal can be accepted here.
	literals          map[rune][]*stdNdfaState				// single-character transitions
	ranges            map[CharacterRange][]*stdNdfaState	// character-range transition
	epsilons          []*stdNdfaState						// epsilon transitions
	acceptTransitions map[string]*stdNdfaState	// accepted terminal -> next state map.
}

type dfaTarget struct {
	c      rune				// The least character in this target interval.
	openGroup  int			// Currently unused
	closeGroup int			// Currently unused
	nxt        *stdDfaState // The next state to transition to.
}

type dfaTargetList []dfaTarget

type stdDfaState struct {
	id        int			// The state id
	accept    int			// True iff the state may accept a terminal
	acceptNxt *stdDfaState	// The next state reached on accepting a terminal
	targets   dfaTargetList	// sort.Interface sortable list of transition target intervals
}

func newStdNdfaState() *stdNdfaState {
	return &stdNdfaState{
		literals: make(map[rune][]*stdNdfaState),
		ranges:   make(map[CharacterRange][]*stdNdfaState),
	}
}

func (st *stdNdfaState) ID() int {
	return st.id
}

// The following accessors just copy their state's fields.
func (st *stdNdfaState) Literals() []rune {
	res := make([]rune, 0, len(st.literals))
	for c := range st.literals {
		res = append(res, c)
	}
	return res
}

func (st *stdNdfaState) Ranges() []CharacterRange {
	res := make([]CharacterRange, 0, len(st.ranges))
	for r := range st.ranges {
		res = append(res, r)
	}
	return res
}

func (st *stdNdfaState) LiteralTransitions(c rune) []NdfaState {
	if m, has := st.literals[c]; has {
		res := make([]NdfaState, len(m))
		for i, k := range m {
			res[i] = k
		}
		return res
	}
	return []NdfaState{}
}

func (st *stdNdfaState) RangeTransitions(r CharacterRange) []NdfaState {
	if m, has := st.ranges[r]; has {
		res := make([]NdfaState, len(m))
		for i, k := range m {
			res[i] = k
		}
		return res
	}
	return []NdfaState{}
}

func (st *stdNdfaState) EpsilonTransitions() []NdfaState {
	res := make([]NdfaState, len(st.epsilons))
	for i, k := range st.epsilons {
		res[i] = k
	}
	return res
}

// This Query() does not need to be perforant; lexers will use the Dfa
// instead.  This query is only for debugging / error unwinding.
func (st *stdNdfaState) Query(c rune) []NdfaState {
	var res []NdfaState
	// Query the literals.
	resmap := make(map[*stdNdfaState]bool)
	if ns, has := st.literals[c]; has {
		for _, s := range ns {
			resmap[s] = true
		}
	}
	// Query the ranges.
	for r, ns := range st.ranges {
		if c >= r.Least() && c <= r.Greatest() {
			for _, s := range ns {
				resmap[s] = true
			}
		}
	}
	// Return the combined result set.
	for ns := range resmap {
		res = append(res, ns)
	}
	return res
}

// clone* functions support subclassers of the lexl package interfaces -
// whenever concrete functionality / access not available through 
// these interfaces are needed, the corresponding objects are cloned
// to std* so the operations can be performed.
func cloneNdfaState(state NdfaState) (NdfaState, error) {
	return nil, errors.New("cloneNdfaState() unimplemented")
}

func cloneNdfaStates(states []*stdNdfaState) ([]*stdNdfaState, error) {
	// Temporarily renumber the id fields for sanity checking and renumbering
	// in the final clone.  First preserve the old ids.
	savedIndex := make([]int, len(states))
	for i, s := range states {
		savedIndex[i] = s.id
		s.id = i
	}
	// No matter what, restore these values by defer so that we do not
	// mutate the states in the input.
	defer func() {
		for i, s := range states {
			s.id = savedIndex[i]
		}
	}()
	// Create the new states.
	res := make([]*stdNdfaState, len(states))
	for i := 0; i < len(res); i++ {
		res[i] = &stdNdfaState{
			id:       i,
			literals: make(map[rune][]*stdNdfaState),
			ranges:   make(map[CharacterRange][]*stdNdfaState),
		}
	}
	// Copy the transitions to the new states, using the canonical new values 
	// instead of the references to items in thegraph we are copying.
	// Since we have renumbered the id fields to match the res[] array indexes,
	// we can just use this slice to look up the values.  If a reference to an
	// state was not in the given extent slice, the referenced object will not be
	// identical to the one in the res[] slice.  In this case, fail with error.
	for i, st := range states {
		for c, m := range st.literals {
			res[i].literals[c] = make([]*stdNdfaState, 0, len(m))
			for _, st := range m {
				if st.id < 0 || st.id >= len(states) || states[st.id] != st {
					return nil, errors.New("referenced state (literal) not within extent")
				}
				res[i].literals[c] = append(res[i].literals[c], res[st.id])
			}
		}
		for r, m := range st.ranges {
			nr := &characterRange{least: r.Least(), greatest: r.Greatest()}
			res[i].ranges[nr] = make([]*stdNdfaState, 0, len(m))
			for _, st := range m {
				if st.id < 0 || st.id >= len(states) || states[st.id] != st {
					return nil, errors.New("referenced state (range) not within extent")
				}
				res[i].ranges[nr] = append(res[i].ranges[nr], res[st.id])
			}
		}
		for _, m := range st.epsilons {
			if m.id < 0 || m.id >= len(states) || states[m.id] != m {
				return nil, errors.New("referenced state (epsilon) not within extent")
			}
			res[i].epsilons = append(res[i].epsilons, res[m.id])
		}
		res[i].accepting = st.accepting
		if st.acceptTransitions != nil {
			res[i].acceptTransitions = make(map[string]*stdNdfaState)
			for k, v := range st.acceptTransitions {
				if v == nil {
					res[i].acceptTransitions[k] = nil
				} else {
					if v.id < 0 || v.id >= len(states) || states[v.id] != v {
						return nil, errors.New("referenced state (accept) not within extent")
					}
					res[i].acceptTransitions[k] = res[v.id]
				}
			}
		}
	}
	return res, nil
}

func cloneDfa(ndfa Dfa) (Dfa, error) {
	return nil, errors.New("cloneDfa() unimplemented")
}

func cloneNdfa(ndfa Ndfa) (Ndfa, error) {
	return nil, errors.New("cloneNdfa() unimplemented")
}

// NdfaStateToString converts a NdfaState to a human-readable multiline
// string representation.  It is intended for debugging.
func NdfaStateToString(ndfaState NdfaState) string {
	var (
		strNdfaState stringable
		ok bool
	)
	if strNdfaState, ok = ndfaState.(stringable); !ok {
		ndfaStateIf, err := cloneNdfaState(ndfaState)
		if err != nil {
			panic(err.Error())
		}
		return ndfaStateIf.(stringable).ToString()
	} 
	return strNdfaState.ToString()
}

// NdfaToString converts an entire Ndfa to a human-readable multiline
// string representation.  It is intended for debugging.
func NdfaToString(ndfa Ndfa) string {
	var (
		strNdfa stringable
		ok bool
	)
	if strNdfa, ok = ndfa.(stringable); !ok {
		ndfaIf, err := cloneNdfa(ndfa)
		if err != nil {
			panic(err.Error())
		}
		return ndfaIf.(stringable).ToString()
	}
	return strNdfa.ToString()
}

func (ndfa *stdNdfa) NumStates() int {
	return len(ndfa.states)
}

func (ndfa *stdNdfa) State(idx int) NdfaState {
	if idx < 0 || idx >= len(ndfa.states) {
		return nil
	}
	return ndfa.states[idx]
}

// DfaItem is an intermediate representation of a DFA state that defined the
// mapping between NDFA states and DFA states.  It may be used to provide debugging
// information to a lexer so that user-friendly error messages may be generated (
// this is curretly unimplemented).
type DfaItem struct {
	id          int						// state id
	states      map[int]int				// the set of NDFA states covered by this DFA state
	hc          uint32					// cached hashcode
	openGroups  map[rune]int			// currently unused
	closeGroups map[rune]int			// currently unused
	accepts     map[int]*DfaItem		// possibly ambiguous list of accepted terminals
	literals    map[rune]*DfaItem		// literal character transition map
	ranges      map[CharacterRange]*DfaItem 	// character range transition map
}

// HashCode (parser.Hashable)
func (di *DfaItem) HashCode() uint32 {
	if di.hc == 0 {
		ids := make([]int, 0, len(di.states))
		for stateID := range di.states {
			ids = append(ids, stateID)
		}
		sort.Ints(ids)
		for _, k := range ids {
			di.hc = (di.hc << 11) | (di.hc >> 21)
			di.hc ^= uint32(k)
		}
	}
	return di.hc
}

// Equals (parser.Hashable)
func (di *DfaItem) Equals(v interface{}) bool {
	if item, ok := v.(*DfaItem); ok {
		if len(item.states) != len(di.states) {
			return false
		}
		for k, v := range item.states {
			v2, has := di.states[k]
			if !has || v != v2 {
				return false
			}
		}
	}
	return true
}

func newDfaItem() *DfaItem {
	return &DfaItem{
		states:      make(map[int]int),
		openGroups:  make(map[rune]int),
		closeGroups: make(map[rune]int),
		accepts:     make(map[int]*DfaItem),
		literals:    make(map[rune]*DfaItem),
		ranges:      make(map[CharacterRange]*DfaItem),
	}
}

func (ndfa *stdNdfa) TransformToDfa() (Dfa, error) {
	terminalIndex := make(map[string]int)		// terminal name -> dfa code
	terminals := []string{}						// slice of all term names indexed by code

	// XXX - Debugging only, remove.
	cle := &stdLexlCharacterLiteralExpression{}
	mapChar := func(r rune) string {
		var buf []rune
		return string(cle.appendClassChar(buf, r))
	}

	// First, precompute the local epsilon closures of each state.
	epsClosures := make([]map[int]int, len(ndfa.states))
	for i := 0; i < len(ndfa.states); i++ {
		epsMap := make(map[int]int)		// keys are the result state id set
		nxt := make(map[int]int)		// keys are the states we still need to recurse on
		nxt[i] = 0
		for len(nxt) > 0 {				// iterate until the "stack" is empty
			var stid, depth int
			
			// grab a single k,v from the map via range
			for k, v := range nxt{		
				fmt.Printf("%d <- %d\n", i, k)
				stid, depth = k, v
				break
			}
			delete(nxt, stid)
			
			// If it's already in the result set, ignore it here.
			if _, has := epsMap[stid]; has {
				continue
			}
			
			// Put it in the result set, and enqueue any states reached through
			// its eps transitions that aren't already traversed.  Since the 
			// "stack" is a map, we don't double-book them, ever.
			epsMap[stid] = depth
			state := ndfa.states[stid]
			for _, nxtState := range state.epsilons {
				if _, has := epsMap[nxtState.id]; !has {
					nxt[nxtState.id] = depth + 1
				}
			}
		}
		fmt.Printf("maplen %d %d\n", i, len(epsMap))
		// Store the result.
		epsClosures[i] = epsMap
	}

	itemIndex := make(map[uint32][]*DfaItem)		// items by hashcode
	// getIndex() uses itemIndex to canonicalize new items, based on their
	// parser.Hashable implementation.
	getIndex := func(item *DfaItem) (*DfaItem, bool) {
		m, has := itemIndex[item.HashCode()]
		if !has {
			return nil, false
		}
		for _, it := range m {
			if it.Equals(item) {
				return it, true
			}
		}
		return nil, false
	}
	
	// addIndex() adds a new item to the canonical map.
	addIndex := func(item *DfaItem) {
		hc := item.HashCode()
		if m, has := itemIndex[hc]; has {
			itemIndex[hc] = append(m, item)
		} else {
			itemIndex[hc] = []*DfaItem{item}
		}
	}
	
	// Create the initial item - this always corresponds to epsilon
	// closure of the zero-indexed state, which is the unique Ndfa entry
	// point by convention.
	initialItem := newDfaItem()
	initialItem.states = epsClosures[0]
	itemIndex[initialItem.HashCode()] = []*DfaItem{initialItem}
	
	items := []*DfaItem{initialItem}	// Completed items 
	stack := []*DfaItem{initialItem}	// Items whose Ndfa set is determined by whose
										// transitions still need to be written.
	// Until the stack is empty...
	for len(stack) > 0 {
		
		// Get a value from the stack.
		cs := stack[len(stack)-1]
		stack = stack[0 : len(stack)-1]
		
		// We will iterate over the Ndfa states in the item's state set, and
		// construct the Dfa transitions by creating prototype items from the unions
		// of closures of each transition's next-state-set, then joining these
		// into non-overlapping intervals by splitting the regions into 
		// non-overlapping partitions and merging the items sharing partition
		// segments together before canonicalizing them through the item index.
		//
		// Deep breath.
		for idx := range cs.states {
			state := ndfa.states[idx]
			fmt.Printf("processing state %d\n", idx)
			if len(state.literals) > 0 {
				for c, m := range state.literals {
					fmt.Printf("   literal %s\n", c)
					nxtItem := newDfaItem()
					for _, ns := range m {
						for k, v := range epsClosures[ns.id] {
							nxtItem.states[k] = v
						}
					}
					cItem, has := getIndex(nxtItem)
					if !has {
						cItem = nxtItem
						addIndex(cItem)
						cItem.id = len(items)
						items = append(items, cItem)
						stack = append(stack, cItem)
					}
					cs.literals[c] = cItem
				}
			}
			if len(state.ranges) > 0 {
				for r, m := range state.ranges {
					fmt.Printf("   range [%s-%s]\n", r.Least(), r.Greatest())
					nxtItem := newDfaItem()
					for _, ns := range m {
						for k, v := range epsClosures[ns.id] {
							nxtItem.states[k] = v
						}
					}
					cItem, has := getIndex(nxtItem)
					if !has {
						cItem = nxtItem
						addIndex(cItem)
						cItem.id = len(items)
						items = append(items, cItem)
						stack = append(stack, cItem)
					}
					cs.ranges[r] = cItem
				}
			}
			if len(state.acceptTransitions) > 0 {
				for name, nxt := range state.acceptTransitions {
					nxtItem := newDfaItem()
					for k, v := range epsClosures[nxt.id] {
						nxtItem.states[k] = v
					}
					cItem, has := getIndex(nxtItem)
					if !has {
						cItem = nxtItem
						addIndex(cItem)
						cItem.id = len(items)
						items = append(items, cItem)
						stack = append(stack, cItem)
					}
					if _, has := terminalIndex[name]; !has {
						terminalIndex[name] = len(terminalIndex)
						terminals = append(terminals, name)
						fmt.Printf("accept %s is %d\n", name, terminalIndex[name])
					}
					cs.accepts[terminalIndex[name]] = cItem
				}
			}
		}
	}
	for i := 0; i < len(epsClosures); i++ {
		fmt.Printf("   *%d = {", i)
		set := []int{}
		for k := range epsClosures[i] {
			set = append(set, k)
		}
		sort.Ints(set)
		for i, k := range set {
			fmt.Printf("%d", k)
			if i < len(set)-1 {
				fmt.Printf(",")
			}
		}
		fmt.Println("}")
	}
	fmt.Printf("generated %d DFA items\n", len(items))
	for _, item := range items {
		fmt.Println(item.ToString())
	}

	// Translate the items into the DFA graph by combining/splitting literals
	// and ranges into regions and discarding state number information.
	graph := make([]stdDfaState, len(items))
	for i := 0; i < len(graph); i++ {
		graph[i].id = i
	}

	for i, item := range items {
		fmt.Printf("DFA STATE: %d\n", i)
		state := &graph[i]
		state.id = i
		state.targets = dfaTargetList(make([]dfaTarget, 0, len(item.literals)+len(item.ranges)))
		for c, nxt := range item.literals {
			fmt.Printf("   add literal %s\n", mapChar(c))
			target := dfaTarget{
				c:   c,
				nxt: &graph[nxt.id],
			}
			state.targets = append(state.targets, target)
		}
		for r, nxt := range item.ranges {
			fmt.Printf("   add range [%s-%s] -> %d\n", mapChar(r.Least()), mapChar(r.Greatest()), nxt.id)
			target := dfaTarget{
				c:   r.Least(),
				nxt: &graph[nxt.id],
			}
			state.targets = append(state.targets, target)
		}
		if len(state.targets) != 0 {
			sort.Sort(state.targets)
			if state.targets[0].c > 0 {
				// Add an initial sentinel.
				state.targets = append(state.targets, dfaTarget{})
				copy(state.targets[1:], state.targets[0:len(state.targets)-1])
				state.targets[0] = dfaTarget{
					c:          -1,
					nxt:        nil,
					//accept:     -1,
					openGroup:  0,
					closeGroup: 0,
				}
				fmt.Println("   add sentinel")
			}
			// Run through the targets and fill in any gaps with non-accepting
			// targets so that we can look at only the target list query result
			// and its successor to match a character.
			var addTargets []*dfaTarget
			for i, st := range state.targets {
				if st.c < 0 {
					fmt.Println("   skip initial interval")
					continue
				}
				if i < len(state.targets)-1 {
					fmt.Printf("   consider interval at %s\n", mapChar(st.c))
					// There is a next target.
					nxtSt := state.targets[i+1]
					if _, has := item.literals[st.c]; has {
						// This target came from a literal.
						fmt.Println("      it is a literal")
						if nxtSt.c > st.c+1 {
							newTarget := &dfaTarget{
								c:      st.c + 1,
								//accept: -1,
							}
							addTargets = append(addTargets, newTarget)
						}
					} else {
						// This target came from a range, which we must search for.
						for r := range item.ranges {
							fmt.Println("      it is a range")
							if r.Least() == st.c {
								fmt.Printf("      found: range [%s-%s]\n", mapChar(r.Least()), mapChar(r.Greatest()))
								if nxtSt.c > r.Greatest()+1 {
									newTarget := &dfaTarget{
										c:      r.Greatest() + 1,
										//accept: -1,
									}
									addTargets = append(addTargets, newTarget)
								}
								break
							}
						}
					}
				} else {
					// We are at the end of the target list.
					skipEndpoint := false
					for r := range item.ranges {
						if r.Least() == st.c {
							if r.Greatest() < 0 {
								skipEndpoint = true
							}
							break
						}
					}
					// If there isn't already a final interval, add one.
					if !skipEndpoint {
						fmt.Println("    add final interval")
						addTargets = append(addTargets, &dfaTarget{
							c:      st.c + 1,
							//accept: -1,
						})
					}
				}
			}
			// Add the new targets back into the state.
			for _, nt := range addTargets {
				state.targets = append(state.targets, *nt)
			}
			sort.Sort(state.targets)
		}
		// Set the proper reduction for this state if available.
		//
		// XXX - Support ambiguity here.   Verify that it's always an OK
		// default to take the first accepting terminal in the state as it
		// appears in the grammar, and put some facility in the ndfa interface
		// for getting at that info.
		if len(item.accepts) > 1 {
			panic("ambiguous terminal reduction currently unimplemented")
		}
		if len(item.accepts) == 0 {
			state.accept = -1
		} else {
			for k, v := range item.accepts {
				state.accept = k
				state.acceptNxt = &graph[v.id]
			}
		}
		fmt.Println(len(state.targets))
		fmt.Println(state.ToString())
	}
	return &stdDfa{
		dfa:       graph,
		terminals: terminals,
	}, nil
}

/*
type dfaTarget struct {
	c          rune
	accept     int
	openGroup  int
	closeGroup int
	nxt        *stdDfaState
}

type dfaTargetList []dfaTarget

type stdDfaState struct {
	id      int
	targets dfaTargetList
}
*/

func (dtl dfaTargetList) Len() int           { return len(dtl) }
func (dtl dfaTargetList) Less(i, j int) bool { return dtl[i].c < dtl[j].c }
func (dtl dfaTargetList) Swap(i, j int)      { dtl[i], dtl[j] = dtl[j], dtl[i] }

func (dtl dfaTargetList) Query(c rune) (*dfaTarget, int) {
	fmt.Printf("Query char >%c:%d<\n", c, c)
	idx := sort.Search(len(dtl), func(x int) bool {
		fmt.Printf("cmp @%d,  %d %d\n", x, c, dtl[x].c)
		return c <= dtl[x].c
	})
	fmt.Printf("lookup %d/%d\n", idx, len(dtl))
	if idx == len(dtl) {
		return nil, idx
	}
	if c != dtl[idx].c {
		if idx > 0 {
			idx--
		}
	}
	if dtl[idx].nxt == nil {
		return nil, idx
	}
	fmt.Printf("result: %d\n", dtl[idx].nxt.id)
	return &dtl[idx], idx
}

// ToString (parser.stringable)
func (di *DfaItem) ToString() string {
	var buf []byte
	cle := &stdLexlCharacterLiteralExpression{}
	mapChar := func(r rune) string {
		var buf []rune
		return string(cle.appendClassChar(buf, r))
	}
	mapRange := func(r CharacterRange) string {
		var buf []rune
		if r.Least() < 0 {
			buf = append(buf, []rune("[-")...)
			buf = cle.appendClassChar(buf, r.Greatest())
			return string(append(buf, ']'))
		} else if r.Greatest() < 0 {
			buf = append(buf, []rune("[")...)
			buf = cle.appendClassChar(buf, r.Least())
			return string(append(buf, []rune("-]")...))
		} else {
			buf = append(buf, '[')
			buf = cle.appendClassChar(buf, r.Least())
			buf = append(buf, '-')
			buf = cle.appendClassChar(buf, r.Greatest())
			return string(append(buf, ']'))
		}
	}
	buf = append(buf, fmt.Sprintf("<%d>:{", di.id)...)
	set := []int{}
	for k := range di.states {
		set = append(set, k)
	}
	sort.Ints(set)
	for i, k := range set {
		buf = append(buf, fmt.Sprintf("%d", k)...)
		if i < len(set)-1 {
			buf = append(buf, ',')
		}
	}
	buf = append(buf, "}\n"...)
	for c, ndi := range di.literals {
		buf = append(buf, fmt.Sprintf("     %s => <%d>\n", mapChar(c), ndi.id)...)
	}
	for r, ndi := range di.ranges {
		buf = append(buf, fmt.Sprintf("     %s => <%d>\n", mapRange(r), ndi.id)...)
	}
	for acc, ndi := range di.accepts {
		buf = append(buf, fmt.Sprintf("     (accept %d) => <%d>\n", acc, ndi.id)...)
	}
	return string(buf)
}

func (ndfa *stdNdfa) ToString() string {
	var buf []rune
	cle := &stdLexlCharacterLiteralExpression{}
	mapChar := func(r rune) string {
		var buf []rune
		return string(cle.appendClassChar(buf, r))
	}
	mapRange := func(r CharacterRange) string {
		var buf []rune
		if r.Least() < 0 {
			buf = append(buf, []rune("[-")...)
			buf = cle.appendClassChar(buf, r.Greatest())
			return string(append(buf, ']'))
		} else if r.Greatest() < 0 {
			buf = append(buf, []rune("[")...)
			buf = cle.appendClassChar(buf, r.Least())
			return string(append(buf, []rune("-]")...))
		} else {
			buf = append(buf, '[')
			buf = cle.appendClassChar(buf, r.Least())
			buf = append(buf, '-')
			buf = cle.appendClassChar(buf, r.Greatest())
			return string(append(buf, ']'))
		}
	}
	for i, st := range ndfa.states {
		buf = append(buf, []rune(fmt.Sprintf("[[%d]]: \n", i))...)
		for c, m := range st.literals {
			if len(m) == 1 {
				buf = append(buf, []rune(fmt.Sprintf("     %s => [[%d]]\n", mapChar(c), m[0].id))...)
			} else {
				buf = append(buf, []rune(fmt.Sprintf("     %s =>\n", mapChar(c)))...)
				for _, mopt := range m {
					buf = append(buf, []rune(fmt.Sprintf("           [[%d]]\n", mopt.id))...)
				}
			}
		}
		for r, m := range st.ranges {
			if len(m) == 1 {
				buf = append(buf, []rune(fmt.Sprintf("     %s => [[%d]]\n", mapRange(r), m[0].id))...)
			} else {
				buf = append(buf, []rune(fmt.Sprintf("     %s =>\n", mapRange(r)))...)
				for _, mopt := range m {
					buf = append(buf, []rune(fmt.Sprintf("           [[%d]]\n", mopt.id))...)
				}
			}
		}
		if len(st.epsilons) > 0 {
			if len(st.epsilons) == 1 {
				buf = append(buf, []rune(fmt.Sprintf("     `e => [[%d]]\n", st.epsilons[0].id))...)
			} else {
				buf = append(buf, []rune("     `e =>\n")...)
				for _, mopt := range st.epsilons {
					buf = append(buf, []rune(fmt.Sprintf("           [[%d]]\n", mopt.id))...)
				}
			}
		}
		if st.accepting {
			if len(st.acceptTransitions) == 0 {
				panic("accepting state with no accepting transitions!")
			}
			for k, v := range st.acceptTransitions {
				buf = append(buf, []rune(fmt.Sprintf("     (accept %s) => [[%d]]\n", k, v.id))...)
			}
		} else {
			if len(st.acceptTransitions) != 0 {
				panic("state with accepting transitions not marked accepting")
			}
		}
	}
	return string(buf)
}

// DfaToString converts an entire Dfa to a human-readable multiline
// string representation.  It is intended for debugging.
func DfaToString(dfa Dfa) string {
	var (
		strDfa stringable
		ok bool
	)
	if strDfa, ok = dfa.(stringable); !ok {
		dfaIf, err := cloneDfa(dfa)
		if err != nil {
			panic(err.Error())
		}
		return dfaIf.(stringable).ToString()
	}
	return strDfa.ToString()
}

func (slds *stdDfaState) ToString() string {
	var buf []byte
	cle := &stdLexlCharacterLiteralExpression{}
	mapChar := func(r rune) string {
		var buf []rune
		return string(cle.appendClassChar(buf, r))
	}
	buf = append(buf, fmt.Sprintf("[%d]\n", slds.id)...)
	for _, target := range slds.targets {
		if target.nxt == nil {
			buf = append(buf, fmt.Sprintf("     %s X\n", mapChar(target.c))...)
		} else {
			buf = append(buf, fmt.Sprintf("     %s [%d]\n", mapChar(target.c), target.nxt.id)...)
		}
	}
	if slds.accept >= 0 {
		buf = append(buf, fmt.Sprintf("     (accept %d) [%d]\n", slds.accept,slds.acceptNxt.id)...)
	}
	return string(buf)
}

func (slds *stdDfa) NumStates() int {
	return len(slds.dfa)
}

func (slds *stdDfa) State(idx int) DfaState {
	if idx < 0 || idx >= len(slds.dfa) {
		return nil
	}
	return &slds.dfa[idx]
}

func (slds *stdDfa) NumTerminals() int {
	return len(slds.terminals)
}

func (slds *stdDfa) Terminal(idx int) string {
	return slds.terminals[idx]
}

func (slds *stdDfa) ToString() string {
	var buf []byte
	for i, terminal := range slds.terminals {
		buf = append(buf, fmt.Sprintf("%d:%s\n", i, terminal)...)
	}
	for _, state := range slds.dfa {
		buf = append(buf, state.ToString()...)
	}
	buf = append(buf, '\n')
	return string(buf)
}

func (slds *stdDfaState) ID() int {
	return slds.id
}

func (slds *stdDfaState) NumIntervals() int {
	return len(slds.targets)
}

func (slds *stdDfaState) IntervalLower(idx int) rune {
	return slds.targets[idx].c
}

func (slds *stdDfaState) IntervalTransition(idx int) DfaState {
	return slds.targets[idx].nxt
}

func (slds *stdDfaState) CanAccept() bool {
	return slds.accept >= 0 && slds.acceptNxt != nil
}

func (slds *stdDfaState) AcceptTransition() (int, DfaState) {
	return slds.accept, slds.acceptNxt
}

//func (dtl dfaTargetList) Query(c rune) (*dfaTarget, int) {

func (slds *stdDfaState) Query(c rune) DfaState {
	if slds == nil {
		panic("WTF2")
	}
	st, _ := slds.targets.Query(c)
	if st == nil {
		return nil
	}
	return st.nxt
}
