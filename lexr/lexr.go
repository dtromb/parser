package lexr

import (
	"bufio"
	"math"
	"strconv"
	"sort"
	"errors"
	"os"
	"fmt"
	"io"
	"github.com/dtromb/parser"
)

type Domain interface {
	NumBlocks() int
	Block(idx int) Block
	Grammar() parser.Grammar
	GenerateNdfas() ([]Ndfa,error)
}

type Block interface {
	Domain() Domain
	Name() string
	Index() int
	NumTermdefs() int
	Termdef(idx int) Termdef
	NumInclusions() int
	Inclusion(idx int) Block
	Ignore() Expression
	HasDefaultForward() bool
	DefaultForward() Block
}

type Termdef interface {
	Block() Block
	Index() int
	Terminal() parser.Term
	Expression() Expression
	NextBlock() Block
	HasNextBlock() bool
}

type Expression interface {
	Type() ExpressionType
	TestMatch(testString string) (bool,[]int)
}

type ExpressionType uint8
const (
	// MatchNever never matches.
	MatchNever ExpressionType = iota
	// MatchAlways always matches.
	MatchAlways
	// MatchCharacterLiteral matches iff the next character is a specified literal character.
	MatchCharacterLiteral
	// MatchStart matches iff the lexer is at the start-of-stream position.
	MatchStart
	// LexlMatchEnd matches iff the lexer is at the end-of-stream position
	LexlMatchEnd
	// MatchSubmatch matches iff the specified submatch matches.  It generates a capturing group which
	// may be later accessed through the recognized token representation.
	MatchSubmatch
	// MatchOptional always matches, consuming the input of a specified submatch if that submatch matches.
	MatchOptional
	// MatchStar always matches, consuming the maximum sequence of repeated inputs to the specified
	// submatch, as long as it continues to match.
	MatchStar
	// MatchPlus matches iff the specified submatch matches at least once.  It consumes input
	// corresponding to the maximum number of repeated submatches.
	MatchPlus
	// MatchQuantified matches iff a specified number (or range) of matches of a specified submatch succeed.
	MatchQuantified
	// MatchCharset matches iff the next character is in the specified character subset.
	MatchCharset
	// MatchSequence matches iff all of the child submatches match the current input, in order.
	MatchSequence
	// MatchAlternation matches iff exactly one of the child submatches match the current input.
	MatchAlternation
)

type CharacterClass interface {
	Negated() bool
	Literals() []rune
	Ranges() []CharacterRange
}

type CharacterRange interface {
	Least() rune
	Greatest() rune
	Test(c rune) bool
	Hash() uint64
}


func WriteDomainLexr0(out io.Writer, d Domain) {
	writeDomainExpression := func (e Expression) {
		out.Write([]byte{'/'})
		WriteExpression(e, out)
		out.Write([]byte{'/'})
	}
	writeDomainTermdef := func (t Termdef) {
		out.Write([]byte(fmt.Sprintf("    %s ", t.Terminal().Name())))
		writeDomainExpression(t.Expression())
		if t.HasNextBlock() {
			out.Write([]byte(fmt.Sprintf(" {%s}", t.NextBlock().Name())))
		}
		out.Write([]byte{'\n'})
	}
	writeDomainBlock := func(b Block) {
		out.Write([]byte(fmt.Sprintf("%s:{{\n", b.Name())))
		if b.HasDefaultForward() {
			out.Write([]byte(fmt.Sprintf("    {%s}\n", b.DefaultForward().Name())))
		}
		if b.Ignore() != nil && b.Ignore().Type() != MatchNever {
			out.Write([]byte("    _ "))
			writeDomainExpression(b.Ignore())
			out.Write([]byte{'\n'})
		}
		for i := 0; i < b.NumTermdefs(); i++ {
			writeDomainTermdef(b.Termdef(i))
		}
		for i := 0; i < b.NumInclusions(); i++ {
			out.Write([]byte(fmt.Sprintf("    {{%s}}\n", b.Inclusion(i).Name())))
		}
		out.Write([]byte("}}\n"))
	}
	for i := 0; i < d.NumBlocks(); i++ {
		writeDomainBlock(d.Block(i))
	}
} 

///

type stdCharacterClass struct {
	negated  bool
	literals map[rune]rune
	ranges   []characterRange
}

type characterRange struct {
	least rune
	greatest rune
}

func normalizeDomain(d Domain) (Domain,error) {
	sd, ok := d.(*stdDomain)
	if !ok {
		s, err := cloneDomain(d)
		if err != nil {
			return nil, err
		}
		sd = s
	}
	termdefs := make([][]*stdDomainTermdef, len(sd.blocks))
	termdefIndex := make([]map[string]*stdDomainTermdef, len(sd.blocks))
	includes := make([][]*stdDomainBlock, len(sd.blocks))
	includesIndex := make([]map[string]bool, len(sd.blocks))
	ignores := make([]Expression, len(sd.blocks))
	for i := 0; i < len(sd.blocks); i++ {
		termdefs[i] = make([]*stdDomainTermdef, len(sd.blocks[i].termdefs))
		copy(termdefs[i], sd.blocks[i].termdefs)
		termdefIndex[i] = make(map[string]*stdDomainTermdef)
		for j := 0; j < len(termdefs[i]); j++ {
			termdefIndex[i][termdefs[i][j].terminal.Name()] = termdefs[i][j]
		}
		includes[i] = make([]*stdDomainBlock, len(sd.blocks[i].includeBlocks))
		copy(includes[i], sd.blocks[i].includeBlocks)
		includesIndex[i] = make(map[string]bool)
		for j := 0; j < len(includes[i]); j++ {
			includesIndex[i][includes[i][j].name] = true
		}
		ignores[i] = sd.blocks[i].ignoreExpr
	}
	for i := 0; i < len(sd.blocks); i++ {
		for len(includes[i]) > 0 {
			nextInclude := includes[i][0]
			includes[i] = includes[i][1:]
			for _, td := range termdefs[nextInclude.index] {
				if _, has := termdefIndex[i][td.terminal.Name()]; has {
					continue // local override
				}
				termdefs[i] = append(termdefs[i], td)
				termdefIndex[i][td.terminal.Name()] = td
			}
			for _, incl := range includes[nextInclude.index] {
				if _, has := includesIndex[i][incl.name]; has {
					continue
				}
				includes[i] = append(includes[i], incl)
				includesIndex[i][incl.name] = true
			}
			if ignores[i] == nil || ignores[i].Type() == MatchNever {
				if ignores[nextInclude.index] != nil {
					ignores[i] = ignores[nextInclude.index]
				}
			}
		}
	}
	nb, err := OpenDomainBuilder(sd.grammar)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(sd.blocks); i++ {
		nb.Block(sd.blocks[i].Name())
		if ignores[i] != nil {
			nb.Ignore(ignores[i])
		} else {
			nb.Ignore(NeverMatchExpression())
		}
		for j := 0; j < len(termdefs[i]); j++ {
			td := termdefs[i][j]
			nb.Termdef(td.Terminal().Name(), td.Expression())
			if td.HasNextBlock() {
				nb.ToBlock(td.NextBlock().Name())
			} else if sd.blocks[i].HasDefaultForward() {
				nb.ToBlock(sd.blocks[i].DefaultForward().Name())
			} else {
				nb.ToBlock(sd.blocks[i].Name())
			}
		}
	}
	newDomain, err := nb.Build()
	if err != nil {
		return nil, err
	}
	return newDomain, nil
}

type domainBlockNdfaNode struct {
	block Block
	id uint32
	initials []NdfaNode   // all epsilons
}

type stdDomainNdfa struct {
	block Block
	nodes[] NdfaNode
}

func (sdn *stdDomainNdfa) Block() Block {
	return sdn.block
}

func (dbn *domainBlockNdfaNode) Id() uint32 {
	return dbn.id
}

func (dbn *domainBlockNdfaNode) Literals() []rune {
	return []rune{}
}

func (dbn *domainBlockNdfaNode) LiteralTransitions(c rune) []NdfaNode {
	return []NdfaNode{}
}

func (dbn *domainBlockNdfaNode) CharacterRanges() []CharacterRange {
	return []CharacterRange{}
}

func (dbn *domainBlockNdfaNode) CharacterRangeTransitions(cr CharacterRange) []NdfaNode {
	return []NdfaNode{}
}

func (dbn *domainBlockNdfaNode) EpsilonTransitions() []NdfaNode	{
	res := make([]NdfaNode, len(dbn.initials))
	copy(res, dbn.initials)
	return res
}

func (dbn *domainBlockNdfaNode) IsTerminal() bool {
	return false
}

func (dbn *domainBlockNdfaNode) Query(c rune) []NdfaNode {
	return []NdfaNode{}
}

func (d *stdDomain) GenerateNdfas() ([]Ndfa,error) {
	nd, err := normalizeDomain(d)
	if err != nil {
		return nil, err
	}
	//fmt.Println("NORMALIZED DOMAIN:")
	WriteDomainLexr0(os.Stdout, nd)
	ndfas := make([]Ndfa, d.NumBlocks())
	nextId := uint32(100)
	ndfaZeros := make([]*domainBlockNdfaNode, len(ndfas))
	for i := 0; i < len(ndfas); i++ {
		ndfaZeros[i] = &domainBlockNdfaNode{
			id: nextId,
			block: nd.Block(i),
		}
		nextId++
	}
	for i := 0; i < len(ndfas); i++ {
		var nodes []NdfaNode
		var initials []NdfaNode
		block := nd.Block(i)
		nodes = append(nodes, ndfaZeros[i])
		if block.Ignore() != nil && block.Ignore().Type() != MatchNever {
			expr := block.Ignore()
			gen, ok := expr.(NdfaNodeGenerator)
			if !ok {
				return nil, errors.New("block ignore expression does not receive NdfaNodeGenerator")
			}
			exprNodes, _ := gen.GenerateNdfaNodes(nextId)
			nextId = exprNodes[len(exprNodes)-1].Id()+1
			for _, nn := range exprNodes {
				if en, ok := nn.(*expressionNdfaNode); ok {
					en.domain = nd
					en.block = block
					en.ignore = true
					en.completed = true
				}
				nodes = append(nodes, nn)
			}
			initials = append(initials, exprNodes[0])
		}
		for j := 0; j < block.NumTermdefs(); j++ {
			termdef := block.Termdef(j)
			expr := termdef.Expression()
			gen, ok := expr.(NdfaNodeGenerator)			
			if !ok {
				return nil, errors.New("block ignore expression does not receive NdfaNodeGenerator")
			}
			exprNodes, _ := gen.GenerateNdfaNodes(nextId)
			nextId = exprNodes[len(exprNodes)-1].Id()+1
			for _, nn := range exprNodes {
				if en, ok := nn.(*expressionNdfaNode); ok {
					en.domain = nd
					en.block = block
					en.termdef = termdef
					en.completed = true
				}
				nodes = append(nodes, nn)
			}
			initials = append(initials, exprNodes[0])
		}
		ndfaZeros[i].initials = initials
		ndfas[i] = &stdDomainNdfa{
			block: block,
			nodes: nodes,
		}
	}
	return ndfas, nil
}
	
func (dn *stdDomainNdfa) NumNodes() int {
	return len(dn.nodes)
}

func (dn *stdDomainNdfa) Node(idx int) NdfaNode {
	if idx < 0 || idx >= len(dn.nodes) {
		panic("node index out of range")
	}
	return dn.nodes[idx]
}

type dfaTransitionInfo struct {
	lowerBound rune
	upperBound rune
	toStates map[int]NdfaNode
	toStateInfo *dfaStateInfo
}

func (dti *dfaTransitionInfo) clone() *dfaTransitionInfo {
	ndti := &dfaTransitionInfo{
		lowerBound: dti.lowerBound,
		upperBound: dti.upperBound,
		toStates: make(map[int]NdfaNode),
	}
	for k, _ := range dti.toStates {
		ndti.toStates[k] = dti.toStates[k]
	}
	return ndti
}

type stdDfa struct {
	nodes []*stdDfaNode
}

type stdDfaNode struct {
	dfa *stdDfa
	index int
	rangeRights []int
	ranges []CharacterRange
	transitions []*stdDfaNode
	transitionIndex map[uint64]*stdDfaNode
	acceptTerm parser.Term
	acceptNext *stdDfaNode
}

func intSetToString(s map[int]bool) string {
	x := make([]int, len(s))
	z := 0
	for k, _ := range s {
		x[z] = k
		z++
	}
	return intArrayToString(x)
}
func intArrayToString(x []int) string {
	var buf []byte
	buf = append(buf, '[')
	for i, k := range x {
		buf = append(buf, strconv.Itoa(k)...)
		if i < len(x) - 1 {
			buf = append(buf, ',')
		}
	}
	buf = append(buf, ']')
	return string(buf)
}

func (dfa *stdDfa) NumStates() int {
	return len(dfa.nodes)
}

func (dfa *stdDfa) State(idx int) DfaNode {
	return dfa.nodes[idx]
}

func (dn *stdDfaNode) Dfa() Dfa {
	return dn.dfa
}

func (dn *stdDfaNode) Id() int {
	return dn.index
}

func (dn *stdDfaNode) TransitionRange(c rune) CharacterRange {
	//fmt.Printf("TransitionRange(%d)\n",c)
	n := sort.Search(len(dn.rangeRights), func(i int) bool {
		return dn.rangeRights[i] >= int(c)
	})
	if n == len(dn.rangeRights) {
		panic("search value out of range")
	}
	return dn.ranges[n]
}

func (dn *stdDfaNode) TransitionLookup(cr CharacterRange) (DfaNode, bool) {
	if k, has := dn.transitionIndex[cr.Hash()]; has && k != nil {
		return k, true
	}
	return nil, false
}

func (dn *stdDfaNode) TransitionQuery(c rune) (DfaNode, bool) {
	n := sort.Search(len(dn.rangeRights), func(i int) bool {
		return dn.rangeRights[i] >= int(c)
	})
	fmt.Printf("   --- query (%d) with %d -> bucket %d [%d-%d]\n", dn.Id(), c, n, dn.ranges[n].Least(), dn.ranges[n].Greatest())
	if n == len(dn.rangeRights) {
		panic("search value out of range")
	}
	if dn.transitions[n] == nil {
		return nil, false
	}
	return dn.transitions[n], true
}

func (dn *stdDfaNode) IsInitial() bool {
	return dn.index == 0
}

func (dn *stdDfaNode) IsAccepting() bool {
	return dn.acceptTerm != nil
}

func (dn *stdDfaNode) AcceptTerm() (parser.Term, bool) {
	if dn.acceptTerm == nil {
		return nil, false
	}
	return dn.acceptTerm, true
}

func (dn *stdDfaNode) AcceptTermNext() (DfaNode, bool) {
	return dn.acceptNext, dn.IsAccepting() && dn.acceptNext != nil
}

type trleftset []*dfaTransitionInfo

func (ls trleftset) Len() int { return len(ls) }
func (ls trleftset) Less(i, j int) bool { return ls[i].lowerBound < ls[j].lowerBound }
func (ls trleftset) Swap(i, j int) { ls[i], ls[j] = ls[j], ls[i] }

type trrightset []*dfaTransitionInfo

func (ls trrightset) Len() int { return len(ls) }
func (ls trrightset) Less(i, j int) bool { return ls[i].upperBound < ls[j].upperBound }
func (ls trrightset) Swap(i, j int) { ls[i], ls[j] = ls[j], ls[i] }

func eqTSets(a, b map[int]NdfaNode) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		if vb, has := b[k]; !has || va != vb {
			return false
		}
	}
	return true
}

func resolveTransitionsMerging(transitions []*dfaTransitionInfo) ([]*dfaTransitionInfo,error) {
	if len(transitions) == 0 {
		return transitions, nil
	}
	trSet := trleftset(make([]*dfaTransitionInfo, len(transitions)))
	copy(trSet, transitions)
	sort.Sort(trSet)
	res := make([]*dfaTransitionInfo, 1, len(transitions))
	rCollapse := func() {
		if len(res) < 2 {
			return
		}
		//return
		ll := res[len(res)-2]
		lr := res[len(res)-1]
		if ll.upperBound != lr.lowerBound - 1 {
			return
		}
		if eqTSets(ll.toStates, lr.toStates) {
			res = res[0:len(res)-1]
			ll.upperBound = lr.upperBound
		}
	}
	res[0] = transitions[0]
	for i := 1; i < len(trSet); i++ {
		lt := res[len(res)-1]
		rt := trSet[i]
		res = res[0:len(res)-1]
		if lt.upperBound < rt.lowerBound {
			res = append(res,lt)
			res = append(res,rt)
			rCollapse()
			continue
		}
		// lt.upperBound >= rt.lowerBound
		lMid := rt.lowerBound
		uMid := lt.upperBound
		if rt.upperBound < uMid {
			uMid = rt.upperBound
		}
		if lt.lowerBound < lMid {
			lInit := &dfaTransitionInfo{
				lowerBound: lt.lowerBound,
				upperBound: lMid-1,
				toStates: lt.toStates,
			}
			res = append(res, lInit)
			rCollapse()
		}
		mid := &dfaTransitionInfo{
			lowerBound: lMid,
			upperBound: uMid,
			toStates: make(map[int]NdfaNode),
		}
		for s, _ := range lt.toStates {
			mid.toStates[s] = lt.toStates[s]
		}
		for s, _ := range rt.toStates {
			mid.toStates[s] = rt.toStates[s]
		}
		res = append(res, mid)
		rCollapse()
		if mid.upperBound < rt.upperBound {
			rFinal := &dfaTransitionInfo{
				lowerBound: mid.upperBound+1,
				upperBound: rt.upperBound,
				toStates: rt.toStates,
			}
			res = append(res, rFinal)
			rCollapse()
		}
		if mid.upperBound < lt.upperBound {
			rFinal := &dfaTransitionInfo{
				lowerBound: mid.upperBound+1,
				upperBound: lt.upperBound,
				toStates: lt.toStates,
			}
			res = append(res, rFinal)
			rCollapse()
		}
	}
	return res, nil
}

type dfaStateInfo struct {
	id int
	states []int
	hc uint64
	transitions []*dfaTransitionInfo
	canAccept bool
	acceptTerm parser.Term
	forwardToBlock Block
}

func (si *dfaStateInfo) Equals(o *dfaStateInfo) bool {
	if si == o {
		return true
	}
	if si.HashCode() != o.HashCode() {
		return false 
	}
	if len(si.states) != len(o.states) {
		return false
	}
	for i, k := range si.states {
		if o.states[i] != k {
			return false
		}
	}
	return true
}

func (si *dfaStateInfo) HashCode() uint64 {
	if si.hc != 0 {
		return si.hc
	} 
	for _, k := range si.states {
		si.hc = (si.hc << 21) | (si.hc >> 43)
		si.hc ^= uint64(k)
	} 
	return si.hc
}

func ndfaNodeClosure(nn NdfaNode) []NdfaNode {
	cSet := make(map[uint32]NdfaNode)
	stack := []NdfaNode{nn}
	for len(stack) > 0 {
		cn := stack[0]
		stack = stack[1:]
		cSet[cn.Id()] = cn
		for _, tn := range cn.EpsilonTransitions() {
			if _, has := cSet[tn.Id()]; !has {
				cSet[tn.Id()] = tn
				stack = append(stack, tn)
			}
		}
	}
	idx := make([]int, 0, len(cSet)) 
	for _, n := range cSet {
		idx = append(idx, int(n.Id()))
	}
	sort.Ints(idx)
	res := make([]NdfaNode, len(cSet)) 
	for i, k := range idx {
		res[i] = cSet[uint32(k)]
	}
	return res
}

func GenerateDomainDfaFromNdfa(ndfa Ndfa, idOffset int, grammar parser.Grammar) (dfa Dfa, dfaInfo []*dfaStateInfo, err error) {
	canonicalDfaInfos := make(map[uint64][]*dfaStateInfo)
	canonicalizeDfaInfo := func(si *dfaStateInfo) *dfaStateInfo {
		hc := si.HashCode()
		m, has := canonicalDfaInfos[hc]
		if has {
			for _, ci := range m {
				if si.Equals(ci) {
					return ci
				}
			}
			canonicalDfaInfos[hc] = append(m,si)
			return si
		}
		canonicalDfaInfos[hc] = []*dfaStateInfo{si}
		return si
	}
	seenDfaInfo := func(si *dfaStateInfo) bool {
		hc := si.HashCode()
		m, has := canonicalDfaInfos[hc]
		if has {
			for _, ci := range m {
				if si.Equals(ci) {
					return true	
				}
			}
		}
		return false
	}
	
	stateClosures := make(map[uint32][]NdfaNode)
	getClosure := func(n NdfaNode) []NdfaNode {
		cl, has := stateClosures[n.Id()]
		if !has {
			cl = ndfaNodeClosure(n)
			stateClosures[n.Id()] = cl
		}
		return cl
	}
	ndfaNodeIndex := make(map[uint32]NdfaNode)
	ndfaNodeIndex[ndfa.Node(0).Id()] = ndfa.Node(0) 
	stateInfos := make([]*dfaStateInfo, 1, ndfa.NumNodes())
	cl0 := getClosure(ndfa.Node(0))
	stateInfos[0] = &dfaStateInfo{
		states: make([]int, len(cl0)),
	}
	for i, n := range cl0 {
		stateInfos[0].states[i] = int(n.Id())
		ndfaNodeIndex[n.Id()] = n
	}
	sort.Ints(stateInfos[0].states)
	initialState := canonicalizeDfaInfo(stateInfos[0])
	stack := []*dfaStateInfo{initialState}
	var allDfaInfos []*dfaStateInfo
	for len(stack) > 0 {
		ci := stack[0]
		stack = stack[1:]
		var trset []*dfaTransitionInfo
		trset = append(trset, &dfaTransitionInfo{
			lowerBound: 0,
			upperBound: math.MaxInt32,
			toStates: make(map[int]NdfaNode),
		})
		minPri := math.MaxInt64
		var minTermdef Termdef
		for _, stid := range ci.states {
			nn := ndfaNodeIndex[uint32(stid)]
			for _, c := range nn.Literals() {
				newTr := &dfaTransitionInfo{
					lowerBound: c,
					upperBound: c,
					toStates: make(map[int]NdfaNode),
				}
				for _, ctr := range nn.LiteralTransitions(c) {
					clset := getClosure(ctr)
					for _, state := range clset {
						newTr.toStates[int(state.Id())] = state
					}
				}
				trset = append(trset, newTr)
				//fmt.Printf(" -- literal transition on %c, states [", c)
				//for id, _ := range newTr.toStates {
				//	fmt.Printf(" %d ", id)
				//} 
				//fmt.Println("]")
			}
			for _, r := range nn.CharacterRanges() {
				newTr := &dfaTransitionInfo{
					lowerBound: r.Least(),
					upperBound: r.Greatest(),
					toStates: make(map[int]NdfaNode),
				}
				for _, rtr := range nn.CharacterRangeTransitions(r) {
					clset := getClosure(rtr)
					for _, state := range clset {
						newTr.toStates[int(state.Id())] = state
					}
				}
				//fmt.Printf(" -- range transition on %c-%c, states [", r.Least(), r.Greatest())
				//for id, _ := range newTr.toStates {
				//	fmt.Printf(" %d ", id)
				//} 
				//fmt.Println("]")
				trset = append(trset, newTr)
			}
			if nn.IsTerminal() {
				if dnn, ok := nn.(DomainNdfaNode); ok {
					if dnn.IsIgnore() {
						minPri = -1
						minTermdef = nil
					} else {
						if dnn.Termdef().Index() < minPri {
							minPri = dnn.Termdef().Index()
							minTermdef  = dnn.Termdef()
						}
					}
				} else {
					return nil, nil, errors.New("terminal ndfa node was not a DomainNdfaNode")
				}
			}
		}
		trset, err := resolveTransitionsMerging(trset)
		if err != nil {
			return nil, nil, err
		}
		for _, tr := range trset {
			//fmt.Printf(" -- tr %d %d-%d (%c-%c) states [", k, tr.lowerBound, tr.upperBound, tr.lowerBound, tr.upperBound)
			//for id, _ := range tr.toStates {
			//	fmt.Printf(" %d ", id)
			//} 
			//fmt.Println("]")
			if len(tr.toStates) == 0 {
				continue
			}
			tr.toStateInfo = &dfaStateInfo{
				states: make([]int, 0, len(tr.toStates)),
			}
			for k, nn := range tr.toStates {
				// tr.toStateInfo.states[i] = int(nn.Id())
				tr.toStateInfo.states = append(tr.toStateInfo.states, k)
				ndfaNodeIndex[nn.Id()] = nn
			}
			sort.Ints(tr.toStateInfo.states)
			if !seenDfaInfo(tr.toStateInfo) {
				tr.toStateInfo = canonicalizeDfaInfo(tr.toStateInfo)
				stack = append(stack, tr.toStateInfo)
			} else {
				tr.toStateInfo = canonicalizeDfaInfo(tr.toStateInfo)
			}
		}
		ci.transitions = trset
		if minPri < math.MaxInt64 {
			ci.canAccept = true
			if minTermdef != nil {
				ci.acceptTerm = minTermdef.Terminal()
				ci.forwardToBlock = minTermdef.NextBlock()
			} else {
				
			}
		}
		ci.id = len(allDfaInfos)
		allDfaInfos = append(allDfaInfos, ci)
	}
	dfaNodes := make([]*stdDfaNode,len(allDfaInfos))
	dfa = &stdDfa{
		nodes: dfaNodes,
	}
	for i := 0; i < len(allDfaInfos); i++ {
		dfaNodes[i] = &stdDfaNode{
			dfa: dfa.(*stdDfa),
			index: i + idOffset,
			transitionIndex: make(map[uint64]*stdDfaNode),
		}	
	}
	for i, cn := range dfaNodes {
		info := allDfaInfos[i]
		fmt.Printf("     (%d): [", i+idOffset)
		for k, st := range info.states {
			fmt.Printf("[%d]", st)
			if k < len(info.states)-1 {
				fmt.Print(",")
			}
		}
		fmt.Println("]")
		var ranges []*characterRange
		var trs []*stdDfaNode
		tridx := 0 
		var nxtId int
		for tridx < len(info.transitions) {
			tr := info.transitions[tridx]
			rng := &characterRange{tr.lowerBound,tr.upperBound}
			//fmt.Printf("     -- initial range %d-%d %c-%c\n", tr.lowerBound, tr.upperBound, tr.lowerBound, tr.upperBound)
			if tr.toStateInfo == nil {
			//	fmt.Printf("     -- range deadends\n")
				nxtId = -1
			} else {
				nxtId = tr.toStateInfo.id
			//	fmt.Printf("     -- range transitions -> (%d)\n", tr.toStateInfo.id)
			}
			tridx++
			for tridx < len(info.transitions) {
				if (nxtId < 0 && info.transitions[tridx].toStateInfo == nil) ||
				   (nxtId >= 0 && info.transitions[tridx].toStateInfo != nil && info.transitions[tridx].toStateInfo.id == nxtId) {
					rng.greatest = info.transitions[tridx].upperBound
					tridx++
				//	fmt.Printf("     -- extending to %d %c\n", rng.greatest, rng.greatest)
				} else {
					break
				}
			}
			//fmt.Printf("     -- finished\n")
			ranges = append(ranges, rng)
			if nxtId > 0 {
				trs = append(trs, dfaNodes[nxtId])
			} else {
				trs = append(trs, nil)
			}
		}
		cn.rangeRights = make([]int, len(ranges))
		cn.ranges = make([]CharacterRange, len(ranges))
		cn.transitions = make([]*stdDfaNode, len(ranges))
		for i, r := range ranges {
			cn.ranges[i] = r
			cn.rangeRights[i] = int(r.Greatest())
			cn.transitions[i] = trs[i]
			cn.transitionIndex[r.Hash()] = trs[i]
		}
		if info.canAccept {
			if info.acceptTerm != nil {
				cn.acceptTerm = info.acceptTerm
			} else {
				cn.acceptTerm = grammar.Epsilon()
			}
		}
	}
	return dfa, allDfaInfos, nil
}

/*
type dfaStateInfo struct {
	states []int
	hc uint64
	transitions []*dfaTransitionInfo
	canAccept bool
	acceptTerm parser.Term
	forwardToBlock Block
}
type stdDfaNode struct {
	dfa *stdDfa
	index int
	rangeRights []int
	ranges []CharacterRange
	transitions []*stdDfaNode
	transitionIndex map[uint64]*stdDfaNode
	acceptTerm parser.Term
}

type stdDfa struct {
	nodes []*stdDfaNode
}
*/


type lexrLexer struct {
	grammar parser.Grammar
	dfas []Dfa
}

type lexrState struct {
	lexer *lexrLexer
	in *bufio.Reader
	line int
	column int
	position int
	dfaState DfaNode
	la rune
	hasLa bool
	laBytes int
	eof bool
	lastError error
	hasToken bool
	nextToken *lexrToken
}

type lexrToken struct {
	state *lexrState
	fpos int
	lpos int
	fline int
	lline int
	fcol int
	lcol int
	terminal parser.Term
	literal string
}

func CreateLexrLexer(lexrDomain Domain) (parser.Lexer, error) {
	ndfas, err := lexrDomain.GenerateNdfas()
	if err != nil {
		return nil, err
	}
	dfas := make([]Dfa, len(ndfas))
	infos := make([][]*dfaStateInfo, len(ndfas))
	ndfaMap := make(map[string]int)
	for i, ndfa := range ndfas {
		sdn, ok := ndfa.(*stdDomainNdfa)
		if !ok {
			return nil, errors.New("generated ndfa was not a *stdDomainNdfa")
		}
		sdn = sdn
		ndfaMap[sdn.Block().Name()] = i
	}
	offset := 0
	for i, ndfa := range ndfas {
		dfa, info, err := GenerateDomainDfaFromNdfa(ndfa, offset, lexrDomain.Grammar())
		if err != nil {
			return nil, err
		}
		offset += len(info)
		dfas[i] = dfa
		infos[i] = info
	}
	for i, _ := range ndfas {
		dfa := dfas[i]
		for j := 0; j < dfa.NumStates(); j++ {
			dfaState := dfa.State(j)
			if dfaState.IsAccepting() {
				fwdBlock := infos[i][j].forwardToBlock
				if fwdBlock == nil {
					term, _ := dfaState.AcceptTerm()
					if term != term.Grammar().Epsilon() {
						panic("non-ignore accept state without a valid forward block")
					}
					dfaState.(*stdDfaNode).acceptNext = dfas[i].State(0).(*stdDfaNode)
				} else {
					fwdBlockId, ok := ndfaMap[fwdBlock.Name()]
					if !ok {
						return nil, errors.New("unknown forward block '"+fwdBlock.Name()+"' in dfa info for accpting dfa state")
					}
					term, _ := dfaState.AcceptTerm()
					fmt.Printf("(%d) ACCEPT %s -> (%d) {%s}\n", 
						dfaState.Id(),
						term.Name(),
						dfas[fwdBlockId].State(0).Id(),
						fwdBlock.Name())
					dfaState.(*stdDfaNode).acceptNext = dfas[fwdBlockId].State(0).(*stdDfaNode)
				}
			}
		}
	}
	lexer := &lexrLexer{
		grammar: lexrDomain.Grammar(),
		dfas: dfas,
	}
	return lexer, nil
}

func (ll *lexrLexer) Grammar() parser.Grammar {
	return ll.grammar
}

func (ll *lexrLexer) Open(in io.Reader) (parser.LexerState, error) {
	var reader *bufio.Reader
	if br, ok := in.(*bufio.Reader); ok {
		reader = br
	} else {
		reader = bufio.NewReader(in)
	}
	state := &lexrState{
		lexer: ll,
		in: reader,
		line: 1,
		column: 1,
		dfaState: ll.dfas[0].State(0),
	}
	return state, nil
}

func (ls *lexrState) Lexer() parser.Lexer {
	return ls.lexer
}

func (ls *lexrState) Reader() io.Reader {
	return ls.in
}

func (ls *lexrState) ateof() bool {
	if !ls.hasLa {
		ls.peek()
	}
	return ls.eof
}

func (ls *lexrState) peek() rune {
	if !ls.hasLa {
		if ls.eof {
			return rune(0)
		}
		var bytes int
		var err error
		ls.la, bytes, err = ls.in.ReadRune()
		if err != nil {
			ls.eof = true
			ls.lastError = err
			return rune(0)
		}
		ls.hasLa = true		
		ls.laBytes = bytes
	}
	return ls.la
}

func (ls *lexrState) read() rune {
	if ls.eof {
		return rune(0)
	}
	var err error
	var bytes int
	if !ls.hasLa {
		ls.la, bytes, err = ls.in.ReadRune()
		if err != nil {
			ls.eof = true
			ls.lastError = err
			return rune(0)
		}
		ls.hasLa = true	
		ls.laBytes = bytes	
	}
	ls.position += ls.laBytes
	if ls.la == rune('\n') {
		ls.line++
		ls.column = 1
	} else {
		ls.column++
	}
	ls.hasLa = false
	return ls.la
}


func (ls *lexrState) readToken() (bool, error) {
	fpos, fline, fcol := ls.position, ls.line, ls.column
	var buf []rune
	for {
		r := ls.peek()
		fmt.Printf("State (%d) next-read: %c (%d)\n", ls.dfaState.Id(), r, r)
		if r == rune(0) {
			if ls.eof && ls.lastError == io.EOF {
				return false, nil
			}
			if ls.eof {
				return false, ls.lastError
			}
		}
		nn, ok := ls.dfaState.TransitionQuery(r)
		if !ok {
			fmt.Println("  -- no transition")
			// Cannot consume rune; ignore/accept if possible
			if ls.dfaState.IsAccepting() {
				var ok bool
				accept, _ := ls.dfaState.AcceptTerm()
				if accept == accept.Grammar().Epsilon() {
					ls.dfaState, ok = ls.dfaState.AcceptTermNext()
					if !ok {
						panic("invalid AcceptTermNext() result")
					}
					fmt.Println("  -- ignore")
					fpos, fline, fcol = ls.position, ls.line, ls.column
					buf = buf[0:0]
					continue
				}
				ls.hasToken = true
				ls.nextToken = &lexrToken{
					state: ls,
					fpos: fpos,
					lpos: ls.position,
					fline: fline,
					lline: ls.line,
					fcol: fcol,
					lcol: ls.column,
					terminal: accept,
					literal: string(buf),
				}
				fmt.Println("  -- accept "+accept.Name())
				ls.dfaState, _ = ls.dfaState.AcceptTermNext()
				return true, nil
			} else {
				fmt.Println("  -- not accept state; fail")
				// Cannot ignore/accept, and no transition for rune - fail lex.
				ls.eof = true
				ls.lastError = errors.New(fmt.Sprintf("invalid runes at %d:%d(%d)", ls.CurrentLine(), ls.CurrentColumn(), ls.CurrentPosition()))
				return false, ls.lastError
			}
		}
		// Consume the rune and transition.
		fmt.Printf("  --  push rune, new state is %d\n", nn.Id())
		buf = append(buf, ls.read())
		ls.dfaState = nn
	}
}

/*

type lexrToken struct {
	state *lexrState
	fpos int
	lpos int
	fline int
	lline int
	fcol int
	lcol int
	terminal parser.Term
	literal string
}

type Token interface {
	LexerState() LexerState
	FirstPosition() int
	LastPosition() int
	FirstLine() int
	LastLine() int
	FirstColumn() int
	LastColumn() int
	Terminal() Term
	Literal() string
*/

func (ls *lexrState) HasMoreTokens() (bool, error) {
	if !ls.hasToken {
		ok, err := ls.readToken()
		if err != nil {
			if err == io.EOF {
				return false, nil
			}
			return false, err
		}
		return ok, nil
	}
	return true, nil
}

func (ls *lexrState) NextToken() (parser.Token, error) {
	if !ls.hasToken {
		_, err := ls.readToken()
		if err != nil {
			return nil, err
		}
	}
	if ls.hasToken {
		ls.hasToken = false
		return ls.nextToken, nil
	}
	return nil, errors.New("lexrState.readToken() did not produce a token")
}

func (ls *lexrState) CurrentLine() int {
	return ls.line
}

func (ls *lexrState) CurrentColumn() int {
	return ls.column
}

func (ls *lexrState) CurrentPosition() int {
	return ls.position
}


func (lt *lexrToken) LexerState() parser.LexerState {
	return lt.state
}

func (lt *lexrToken) FirstPosition() int {
	return lt.fpos
}

func (lt *lexrToken) LastPosition() int {
	return lt.lpos
}

func (lt *lexrToken) FirstLine() int {
	return lt.fline
}

func (lt *lexrToken) LastLine() int {
	return lt.lline
}

func (lt *lexrToken) FirstColumn() int {
	return lt.fcol
}

func (lt *lexrToken) LastColumn() int {
	return lt.lcol
}

func (lt *lexrToken) Terminal() parser.Term {
	return lt.terminal
}

func (lt *lexrToken) Literal() string {
	return lt.literal
}
