package lexl

import (
	"errors"
	"fmt"
	"sort"
)

type LexlNdfa interface {
	NumStates() int
	State(idx int) LexlNdfaState
	TransformToDfa() (LexlDfa, error)
}

type LexlNdfaState interface {
	Id() int
	Literals() []rune
	Ranges() []CharacterRange
	LiteralTransitions(c rune) []LexlNdfaState
	RangeTransitions(r CharacterRange) []LexlNdfaState
	EpsilonTransitions() []LexlNdfaState
	Query(c rune) []LexlNdfaState
}

type LexlDfa interface {
	NumStates() int
	State(idx int) LexlDfaState
	NumTerminals() int
	Terminal(idx int) string
}

type LexlDfaState interface {
	Id() int
	NumIntervals() int
	IntervalLower(idx int) rune
	IntervalTransition(idx int) LexlDfaState
	CanAccept() bool
	AcceptTransition() (int, LexlDfaState)
	Query(c rune) LexlDfaState
}

///

type ndfaStateGenerator interface {
	GenerateNdfaStates() ([]*stdLexlNdfaState, error)
}

type stdLexlDfa struct {
	dfa       []stdLexlDfaState
	terminals []string
}

type stdLexlNdfa struct {
	states []*stdLexlNdfaState
}

type stdLexlNdfaState struct {
	id                int
	accepting         bool
	literals          map[rune][]*stdLexlNdfaState
	ranges            map[CharacterRange][]*stdLexlNdfaState
	epsilons          []*stdLexlNdfaState
	acceptTransitions map[string]*stdLexlNdfaState
}

type dfaTarget struct {
	c      rune
	accept int // XXX - remove this.   accepting terminals is a function of the dfa state -
	// these targets only define transitions where a character is consumed.
	openGroup  int
	closeGroup int
	nxt        *stdLexlDfaState
}

type dfaTargetList []dfaTarget

type stdLexlDfaState struct {
	id        int
	accept    int
	acceptNxt *stdLexlDfaState
	targets   dfaTargetList
}

func newStdLexlNdfaState() *stdLexlNdfaState {
	return &stdLexlNdfaState{
		literals: make(map[rune][]*stdLexlNdfaState),
		ranges:   make(map[CharacterRange][]*stdLexlNdfaState),
	}
}

func (st *stdLexlNdfaState) Id() int {
	return st.id
}

func (st *stdLexlNdfaState) Literals() []rune {
	res := make([]rune, 0, len(st.literals))
	for c, _ := range st.literals {
		res = append(res, c)
	}
	return res
}

func (st *stdLexlNdfaState) Ranges() []CharacterRange {
	res := make([]CharacterRange, 0, len(st.ranges))
	for r, _ := range st.ranges {
		res = append(res, r)
	}
	return res
}

func (st *stdLexlNdfaState) LiteralTransitions(c rune) []LexlNdfaState {
	if m, has := st.literals[c]; has {
		res := make([]LexlNdfaState, len(m))
		for i, k := range m {
			res[i] = k
		}
		return res
	}
	return []LexlNdfaState{}
}

func (st *stdLexlNdfaState) RangeTransitions(r CharacterRange) []LexlNdfaState {
	if m, has := st.ranges[r]; has {
		res := make([]LexlNdfaState, len(m))
		for i, k := range m {
			res[i] = k
		}
		return res
	}
	return []LexlNdfaState{}
}

func (st *stdLexlNdfaState) EpsilonTransitions() []LexlNdfaState {
	res := make([]LexlNdfaState, len(st.epsilons))
	for i, k := range st.epsilons {
		res[i] = k
	}
	return res
}

func (st *stdLexlNdfaState) Query(c rune) []LexlNdfaState {
	var res []LexlNdfaState
	resmap := make(map[*stdLexlNdfaState]bool)
	if ns, has := st.literals[c]; has {
		for _, s := range ns {
			resmap[s] = true
		}
	}
	for r, ns := range st.ranges {
		if c >= r.Least() && c <= r.Greatest() {
			for _, s := range ns {
				resmap[s] = true
			}
		}
	}
	for ns, _ := range resmap {
		res = append(res, ns)
	}
	return res
}

func cloneNdfaState(state LexlNdfaState) (LexlNdfaState, error) {
	return nil, errors.New("cloneNdfaState() unimplemented")
}

func cloneNdfaStates(states []*stdLexlNdfaState) ([]*stdLexlNdfaState, error) {
	savedIndex := make([]int, len(states))
	for i, s := range states {
		savedIndex[i] = s.id
		s.id = i
	}
	defer func() {
		for i, s := range states {
			s.id = savedIndex[i]
		}
	}()
	res := make([]*stdLexlNdfaState, len(states))
	for i := 0; i < len(res); i++ {
		res[i] = &stdLexlNdfaState{
			id:       i,
			literals: make(map[rune][]*stdLexlNdfaState),
			ranges:   make(map[CharacterRange][]*stdLexlNdfaState),
		}
	}
	for i, st := range states {
		for c, m := range st.literals {
			res[i].literals[c] = make([]*stdLexlNdfaState, 0, len(m))
			for _, st := range m {
				if st.id < 0 || st.id >= len(states) || states[st.id] != st {
					return nil, errors.New("referenced state (literal) not within extent")
				}
				res[i].literals[c] = append(res[i].literals[c], res[st.id])
			}
		}
		for r, m := range st.ranges {
			nr := &characterRange{least: r.Least(), greatest: r.Greatest()}
			res[i].ranges[nr] = make([]*stdLexlNdfaState, 0, len(m))
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
			res[i].acceptTransitions = make(map[string]*stdLexlNdfaState)
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

func cloneDfa(ndfa LexlDfa) (LexlDfa, error) {
	return nil, errors.New("cloneDfa() unimplemented")
}

func cloneNdfa(ndfa LexlNdfa) (LexlNdfa, error) {
	return nil, errors.New("cloneNdfa() unimplemented")
}

func NdfaStateToString(ndfaState LexlNdfaState) string {
	if strNdfaState, ok := ndfaState.(stringable); !ok {
		ndfaStateIf, err := cloneNdfaState(ndfaState)
		if err != nil {
			panic(err.Error())
		}
		return ndfaStateIf.(stringable).ToString()
	} else {
		return strNdfaState.ToString()
	}
}

func NdfaToString(ndfa LexlNdfa) string {
	if strNdfa, ok := ndfa.(stringable); !ok {
		ndfaIf, err := cloneNdfa(ndfa)
		if err != nil {
			panic(err.Error())
		}
		return ndfaIf.(stringable).ToString()
	} else {
		return strNdfa.ToString()
	}
}

func (ndfa *stdLexlNdfa) NumStates() int {
	return len(ndfa.states)
}

func (ndfa *stdLexlNdfa) State(idx int) LexlNdfaState {
	if idx < 0 || idx >= len(ndfa.states) {
		return nil
	}
	return ndfa.states[idx]
}

/*
type dfaTarget struct {
	c   rune
	nxt *stdLexlDfaState
}

type dfaTargetList []dfaTarget

type stdLexlDfaState struct {
	id        int
	hc		  uint32
	targets   dfaTargetList
}
*/

type lexlDfaItem struct {
	id          int
	states      map[int]int
	hc          uint32
	openGroups  map[rune]int
	closeGroups map[rune]int
	accepts     map[int]*lexlDfaItem
	literals    map[rune]*lexlDfaItem
	ranges      map[CharacterRange]*lexlDfaItem
}

func (di *lexlDfaItem) HashCode() uint32 {
	if di.hc == 0 {
		ids := make([]int, 0, len(di.states))
		for stateId, _ := range di.states {
			ids = append(ids, stateId)
		}
		sort.Ints(ids)
		for _, k := range ids {
			di.hc = (di.hc << 11) | (di.hc >> 21)
			di.hc ^= uint32(k)
		}
	}
	return di.hc
}

func (di *lexlDfaItem) Equals(v interface{}) bool {
	if item, ok := v.(*lexlDfaItem); ok {
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

func newLexlDfaItem() *lexlDfaItem {
	return &lexlDfaItem{
		states:      make(map[int]int),
		openGroups:  make(map[rune]int),
		closeGroups: make(map[rune]int),
		accepts:     make(map[int]*lexlDfaItem),
		literals:    make(map[rune]*lexlDfaItem),
		ranges:      make(map[CharacterRange]*lexlDfaItem),
	}
}

func (ndfa *stdLexlNdfa) TransformToDfa() (LexlDfa, error) {
	terminalIndex := make(map[string]int)
	terminals := []string{}

	cle := &stdLexlCharacterLiteralExpression{}
	mapChar := func(r rune) string {
		var buf []rune
		return string(cle.appendClassChar(buf, r))
	}

	// First, precompute the local epsilon closures of each state.
	epsClosures := make([]map[int]int, len(ndfa.states))
	for i := 0; i < len(ndfa.states); i++ {
		epsMap := make(map[int]int)
		nxt := make(map[int]int)
		nxt[i] = 0
		for len(nxt) > 0 {
			var stid, depth int
			for k, v := range nxt {
				fmt.Printf("%d <- %d\n", i, k)
				stid, depth = k, v
				break
			}
			delete(nxt, stid)
			if _, has := epsMap[stid]; has {
				continue
			}
			epsMap[stid] = depth
			state := ndfa.states[stid]
			for _, nxtState := range state.epsilons {
				if _, has := epsMap[nxtState.id]; !has {
					nxt[nxtState.id] = depth + 1
				}
			}
		}
		fmt.Printf("maplen %d %d\n", i, len(epsMap))
		epsClosures[i] = epsMap
	}

	itemIndex := make(map[uint32][]*lexlDfaItem)
	getIndex := func(item *lexlDfaItem) (*lexlDfaItem, bool) {
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
	addIndex := func(item *lexlDfaItem) {
		hc := item.HashCode()
		if m, has := itemIndex[hc]; has {
			itemIndex[hc] = append(m, item)
		} else {
			itemIndex[hc] = []*lexlDfaItem{item}
		}
	}
	initialItem := newLexlDfaItem()
	initialItem.states = epsClosures[0]
	itemIndex[initialItem.HashCode()] = []*lexlDfaItem{initialItem}
	items := []*lexlDfaItem{initialItem}
	stack := []*lexlDfaItem{initialItem}
	for len(stack) > 0 {
		cs := stack[len(stack)-1]
		stack = stack[0 : len(stack)-1]
		for idx, _ := range cs.states {
			state := ndfa.states[idx]
			fmt.Printf("processing state %d\n", idx)
			if len(state.literals) > 0 {
				for c, m := range state.literals {
					fmt.Printf("   literal %s\n", c)
					nxtItem := newLexlDfaItem()
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
					nxtItem := newLexlDfaItem()
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
					nxtItem := newLexlDfaItem()
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
		for k, _ := range epsClosures[i] {
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
	graph := make([]stdLexlDfaState, len(items))
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
					accept:     -1,
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
								accept: -1,
							}
							addTargets = append(addTargets, newTarget)
						}
					} else {
						// This target came from a range, which we must search for.
						for r, _ := range item.ranges {
							fmt.Println("      it is a range")
							if r.Least() == st.c {
								fmt.Printf("      found: range [%s-%s]\n", mapChar(r.Least()), mapChar(r.Greatest()))
								if nxtSt.c > r.Greatest()+1 {
									newTarget := &dfaTarget{
										c:      r.Greatest() + 1,
										accept: -1,
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
					for r, _ := range item.ranges {
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
							accept: -1,
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
	return &stdLexlDfa{
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
	nxt        *stdLexlDfaState
}

type dfaTargetList []dfaTarget

type stdLexlDfaState struct {
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

func (di *lexlDfaItem) ToString() string {
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
	for k, _ := range di.states {
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

func (ndfa *stdLexlNdfa) ToString() string {
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

func DfaToString(dfa LexlDfa) string {
	if strDfa, ok := dfa.(stringable); !ok {
		dfaIf, err := cloneDfa(dfa)
		if err != nil {
			panic(err.Error())
		}
		return dfaIf.(stringable).ToString()
	} else {
		return strDfa.ToString()
	}
}

func (ds *stdLexlDfaState) ToString() string {
	var buf []byte
	cle := &stdLexlCharacterLiteralExpression{}
	mapChar := func(r rune) string {
		var buf []rune
		return string(cle.appendClassChar(buf, r))
	}
	buf = append(buf, fmt.Sprintf("[%d]\n", ds.id)...)
	for _, target := range ds.targets {
		if target.nxt == nil {
			buf = append(buf, fmt.Sprintf("     %s X\n", mapChar(target.c))...)
		} else {
			buf = append(buf, fmt.Sprintf("     %s [%d]\n", mapChar(target.c), target.nxt.id)...)
		}
	}
	if ds.accept >= 0 {
		buf = append(buf, fmt.Sprintf("     (accept %d) [%d]\n", ds.accept, ds.acceptNxt.id)...)
	}
	return string(buf)
}

func (sld *stdLexlDfa) NumStates() int {
	return len(sld.dfa)
}

func (sld *stdLexlDfa) State(idx int) LexlDfaState {
	if idx < 0 || idx >= len(sld.dfa) {
		return nil
	}
	return &sld.dfa[idx]
}

func (sld *stdLexlDfa) NumTerminals() int {
	return len(sld.terminals)
}

func (sld *stdLexlDfa) Terminal(idx int) string {
	return sld.terminals[idx]
}

func (sld *stdLexlDfa) ToString() string {
	var buf []byte
	for i, terminal := range sld.terminals {
		buf = append(buf, fmt.Sprintf("%d:%s\n", i, terminal)...)
	}
	for _, state := range sld.dfa {
		buf = append(buf, state.ToString()...)
	}
	buf = append(buf, '\n')
	return string(buf)
}

func (slds *stdLexlDfaState) Id() int {
	return slds.id
}

func (slds *stdLexlDfaState) NumIntervals() int {
	return len(slds.targets)
}

func (slds *stdLexlDfaState) IntervalLower(idx int) rune {
	return slds.targets[idx].c
}

func (slds *stdLexlDfaState) IntervalTransition(idx int) LexlDfaState {
	return slds.targets[idx].nxt
}

func (slds *stdLexlDfaState) CanAccept() bool {
	return slds.accept >= 0 && slds.acceptNxt != nil
}

func (slds *stdLexlDfaState) AcceptTransition() (int, LexlDfaState) {
	return slds.accept, slds.acceptNxt
}

//func (dtl dfaTargetList) Query(c rune) (*dfaTarget, int) {

func (slds *stdLexlDfaState) Query(c rune) LexlDfaState {
	if slds == nil {
		panic("WTF2")
	}
	st, _ := slds.targets.Query(c)
	if st == nil {
		return nil
	}
	return st.nxt
}
