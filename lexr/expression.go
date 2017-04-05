package lexr

import (
	"reflect"
	"math"
	"github.com/dtromb/parser"
	"strconv"
	"unicode"
	"fmt"
	"io"
	"sort"
)

type expressionNdfaNode struct {
	id uint32
	literals map[rune][]*expressionNdfaNode
	ranges map[uint64][]*expressionNdfaNode
	epsilons []NdfaNode
	terminal parser.Term
	accepting bool
	initial bool
	completed bool
	domain Domain
	termdef Termdef
	block Block
	ignore bool
}

func newExpressionNdfaNode(id uint32) *expressionNdfaNode {
	return &expressionNdfaNode{
		id: id,
		literals: make(map[rune][]*expressionNdfaNode),
		ranges: make(map[uint64][]*expressionNdfaNode),
	}
}

type alwaysMatchExpression struct{}

func AlwaysMatchExpression() Expression {
	return &alwaysMatchExpression{}
}

func (ame *alwaysMatchExpression) Type() ExpressionType {
	return MatchAlways
}

func (ame *alwaysMatchExpression) TestMatch(str string) (bool, []int) {
	if len(str) > 0 {
		return true, []int{1}
	}
	return false, []int{}
}

func (ame *alwaysMatchExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	s := newExpressionNdfaNode(firstId)
	r := newExpressionNdfaNode(firstId+1)
	all := characterRange{0, math.MaxInt32}
	s.initial = true
	s.ranges[all.Hash()] = []*expressionNdfaNode{r}
	r.accepting = true
	return []NdfaNode{s,r}, 1
}

type neverMatchExpression struct {}

func NeverMatchExpression() Expression {
	return &neverMatchExpression{}
}

func (nme *neverMatchExpression) Type() ExpressionType {
	return MatchNever
}

func (nme *neverMatchExpression) TestMatch(str string) (bool, []int) {
	return false, []int{}
}

func (nme *neverMatchExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	s := newExpressionNdfaNode(firstId)
	s.initial = true
	return []NdfaNode{s}, 0
}

type characterLiteralExpression struct {
	literal rune
}

func CharacterLiteralExpression(c rune) Expression {
	return &characterLiteralExpression{literal: c}
}

func (cle *characterLiteralExpression) TestMatch(str string) (bool,[]int) {
	return len(str) > 0 && []rune(str)[0] == cle.literal, []int{1}
}

func (cle *characterLiteralExpression) Type() ExpressionType {
	return MatchCharacterLiteral
}

func (cle *characterLiteralExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	s := newExpressionNdfaNode(firstId)
	r := newExpressionNdfaNode(firstId+1)
	s.initial = true
	r.accepting = true
	s.literals[cle.literal] = []*expressionNdfaNode{r}
	return []NdfaNode{s,r}, 1
}

type characterClassExpression struct {
	class CharacterClass
}

func CharacterClassExpression(cc CharacterClass) Expression {
	return &characterClassExpression{
		class: cc,
	}
}

func (cce *characterClassExpression) Type() ExpressionType {
	return MatchCharset
}

func (cce *characterClassExpression) TestMatch(str string) (bool,[]int) {
	rs := []rune(str)
	if len(rs) == 0 {
		return false, []int{}
	}
	has := false 
	for _, c := range cce.class.Literals() {
		if c == rs[0] {
			has = true 
			break
		}
	}
	if !has {
		for _, r := range cce.class.Ranges() {
			if r.Least() <= rs[0] && r.Greatest() >= rs[0] {
				has = true
				break
			}
		}
	}
	if cce.class.Negated() {
		has = !has
	}
	if has {
		return true, []int{1}
	}
	return false, []int{}
}

func (cce *characterClassExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	s := newExpressionNdfaNode(firstId)
	r := newExpressionNdfaNode(firstId+1)
	s.initial = true
	r.accepting = true
	var intervals []*characterRange
	for _, c := range cce.class.Literals() {
		intervals = append(intervals, &characterRange{c,c})
	}
	for _, r := range cce.class.Ranges() {
		intervals = append(intervals, &characterRange{r.Least(),r.Greatest()})
	}
	for _, iv := range intervals {
		cr := iv
		if cr.least < 0 {
			cr.least = 0
		}
		if cr.greatest < 0 {
			cr.greatest = math.MaxInt32
		}
	}
	intervals = regularizeBoundedIntervals(intervals)
	if cce.class.Negated() {
		intervals = invertRegularizedIntervals(intervals)
	}
	for _, iv := range intervals {
		if iv.Least() == iv.Greatest() {
			s.literals[iv.Least()] = []*expressionNdfaNode{r}
		} else if iv.Least()+1 == iv.Greatest() {
			s.literals[iv.Least()] = []*expressionNdfaNode{r}
			s.literals[iv.Greatest()] = []*expressionNdfaNode{r}
		} else {
			s.ranges[iv.Hash()] = []*expressionNdfaNode{r}
		}
	}
	return []NdfaNode{s,r}, 1
}

type sequenceExpression struct {
	exprs []Expression
}

func SequenceExpression(exprs ...Expression) Expression {
	expr := &sequenceExpression{
		exprs: make([]Expression, 0, len(exprs)),
	}
	for _, e := range exprs {
		expr.exprs = append(expr.exprs, e)
	}
	return expr
}

func (se *sequenceExpression) Type() ExpressionType {
	return MatchSequence
}

func (se *sequenceExpression) TestMatch(str string) (bool,[]int) {
	positions := make(map[int]int)
	positions[0] = 0
	for _, expr := range se.exprs {
		nextPositions := make(map[int]int)
		for _, pos := range positions {
			nstr := str[pos:]
			ok, npos := expr.TestMatch(nstr)
			if ok {
				for _, dp := range npos {
					nextPositions[pos+dp] = pos+dp
				}
			}
		}
		if len(nextPositions) == 0 {
			return false, []int{}
		} 
		positions = nextPositions
	}
	res := make([]int, 0, len(positions))
	for _, p := range positions {
		res[p] = p
	}
	sort.Ints(res)
	return true, res
}

func (se *sequenceExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	nextId := firstId
	exprGen, ok := se.exprs[0].(NdfaNodeGenerator)
	if !ok {
		panic("sequence subexpression type "+reflect.TypeOf(se.exprs[0]).String()+" does not receive NdfaNodeGenerator")
	}
	work, accCount := exprGen.GenerateNdfaNodes(nextId)
	nextId = work[len(work)-1].Id()+1
	var next []NdfaNode
	var nxtAccCount int
	for i := 1; i < len(se.exprs); i++ {
		exprGen, ok = se.exprs[i].(NdfaNodeGenerator)
		if !ok {
			panic("sequence subexpression type "+reflect.TypeOf(se.exprs[0]).String()+" does not receive NdfaNodeGenerator")
		}
		next, nxtAccCount = exprGen.GenerateNdfaNodes(nextId)
		nextId = next[len(next)-1].Id()+1
		newInit, ok := next[0].(*expressionNdfaNode)
		if !ok {
			panic("sequence subexpression node was not an *expressionNdfaNode")
		}
		newInit.initial = false
		for j := len(work)-accCount; j < len(work); j++ {
			oldAcc, ok := work[j].(*expressionNdfaNode);
			if !ok {
				panic("sequence subexpression node was not an *expressionNdfaNode")
			}
			oldAcc.accepting = false
			oldAcc.epsilons = append(oldAcc.epsilons, newInit)
		}
		work = append(work, next...)
	}
	return work, nxtAccCount
}

type alternationExpression struct {
	exprs []Expression
}

func AlternationExpression(exprs ...Expression) Expression {
	expr := &alternationExpression{
		exprs: make([]Expression, len(exprs)),
	}
	copy(expr.exprs, exprs)
	return expr
}

func (ae *alternationExpression) Type() ExpressionType {
	return MatchAlternation
}

func (ae *alternationExpression) TestMatch(str string) (bool, []int) {
	positions := make(map[int]int)
	for _, expr := range ae.exprs {
		ok, lens := expr.TestMatch(str)
		if ok {
			for _, r := range lens {
				positions[r] = r
			}
		}
	}
	if len(positions) == 0 {
		return false, []int{}
	}
	res := make([]int, 0, len(positions))
	for _, p := range positions {
		res = append(res, p)
	}
	sort.Ints(res)
	return true, res
}

func (ae *alternationExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	nextId := firstId
	s := newExpressionNdfaNode(nextId)
	nextId++
	s.initial = true
	work := []NdfaNode{s}
	finals := []NdfaNode{}
	for i := 0; i < len(ae.exprs); i++ {
		exprGen, ok := ae.exprs[i].(NdfaNodeGenerator)
		if !ok {
			panic("alternation subexpression type "+reflect.TypeOf(ae.exprs[0]).String()+" does not receive NdfaNodeGenerator")
		}
		next, nxtAccCount := exprGen.GenerateNdfaNodes(nextId)
		nextId = next[len(next)-1].Id()+1	
		init, ok := next[0].(*expressionNdfaNode)
		if !ok {
			panic("alternation subexpression node was not an *expressionNdfaNode")
		}
		init.initial = false
		s.epsilons = append(s.epsilons, init)
		work = append(work, next[0:len(next)-nxtAccCount]...)
		finals = append(finals, next[len(next)-nxtAccCount:]...)
	}
	accCount := len(finals)
	work = append(work, finals...)
	return work, accCount
}

type plusExpression struct {
	expr Expression
}

func PlusExpression(expr Expression) Expression {
	return &plusExpression{
		expr: expr,
	}
}

func (pe *plusExpression) Type() ExpressionType {
	return MatchPlus
}

func (pe *plusExpression) TestMatch(str string) (bool, []int) {
	positions := make(map[int]int)
	allPositions := make(map[int]int)
	positions[0] = 0
	for {
		nextPositions := make(map[int]int)
		for _, p := range positions {
			nstr := str[p:]
			ok, next := pe.expr.TestMatch(nstr)
			if ok {
				for _, np := range next {
					if _, has := allPositions[np]; !has && np > 0 {
						allPositions[p+np] = p+np
						nextPositions[p+np] = p+np
					}
				}
			}
		}
		if len(nextPositions) == 0 {
			break
		}
		positions = nextPositions
		nextPositions = make(map[int]int)
	}
	if len(allPositions) == 0 {
		return false, []int{}
	}
	res := make([]int, 0, len(allPositions))
	for _, p := range allPositions {
		res[p] = p
	}
	sort.Ints(res)
	return true, res
}

func (pe *plusExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	exprGen, ok := pe.expr.(NdfaNodeGenerator)
	if !ok {
		panic("plus subexpression type "+reflect.TypeOf(pe.expr).String()+" does not receive NdfaNodeGenerator")
	}
	work, accCount := exprGen.GenerateNdfaNodes(firstId)
	r := newExpressionNdfaNode(work[len(work)-1].Id()+1)
	init, ok := work[0].(*expressionNdfaNode)
	if !ok {
		panic("plus subexpression node was not an *expressionNdfaNode")
	}
	r.accepting = true
	for i := len(work)-accCount; i < len(work); i++ {
		acc, ok := work[i].(*expressionNdfaNode)
		if !ok {
			panic("plus subexpression node was not an *expressionNdfaNode")
		}
		acc.accepting = false
		acc.epsilons = append(acc.epsilons, r)
		acc.epsilons = append(acc.epsilons, init)
	}
	work = append(work, r)
	return work, 1
}

type starExpression struct {
	expr Expression
}

func StarExpression(expr Expression) Expression {
	return &starExpression{expr: expr}
}

func (se *starExpression) Type() ExpressionType {
	return MatchStar
}

func (se *starExpression) TestMatch(str string) (bool, []int) {
	positions := make(map[int]int)
	allPositions := make(map[int]int)
	positions[0] = 0
	allPositions[0] = 0
	for {
		nextPositions := make(map[int]int)
		for _, p := range positions {
			nstr := str[p:]
			ok, next := se.expr.TestMatch(nstr)
			if ok {
				for _, np := range next {
					if _, has := allPositions[np]; !has && np > 0 {
						allPositions[p+np] = p+np
						nextPositions[p+np] = p+np
					}
				}
			}
		}
		if len(nextPositions) == 0 {
			break
		}
		positions = nextPositions
		nextPositions = make(map[int]int)
	}
	res := make([]int, 0, len(allPositions))
	for _, p := range allPositions {
		res[p] = p
	}
	sort.Ints(res)
	return true, res
}

func (se *starExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	exprGen, ok := se.expr.(NdfaNodeGenerator)
	if !ok {
		panic("star subexpression type "+reflect.TypeOf(se.expr).String()+" does not receive NdfaNodeGenerator")
	}
	work, accCount := exprGen.GenerateNdfaNodes(firstId)
	r := newExpressionNdfaNode(work[len(work)-1].Id()+1)
	init, ok := work[0].(*expressionNdfaNode)
	init.epsilons = append(init.epsilons, r)
	if !ok {
		panic("star subexpression node was not an *expressionNdfaNode")
	}
	r.accepting = true
	for i := len(work)-accCount; i < len(work); i++ {
		acc, ok := work[i].(*expressionNdfaNode)
		if !ok {
			panic("star subexpression node was not an *expressionNdfaNode")
		}
		acc.accepting = false
		acc.epsilons = append(acc.epsilons, r)
		acc.epsilons = append(acc.epsilons, init)
	}
	work = append(work, r)
	return work, 1
}

type quantifiedExpression struct {
	expr Expression
	min int
	max int
}

func QuantifiedExpression(expr Expression, min, max int) Expression {
	if min < 0 {
		min = 0
	}
	if max > 0 && max < min {
		panic("quantifier maximum may not be positive and less than the minimum")
	}
	if max == 0 {
		panic("quantifier maximum may not be zero")
	}
	return &quantifiedExpression{
		expr: expr,
		min: min,
		max: max,
	}
}

func (qe *quantifiedExpression) Type() ExpressionType {
	return MatchQuantified
}

func (qe *quantifiedExpression) TestMatch(str string) (bool, []int) {
	positions := make(map[int]int)
	positions[0] = 0
	for i := 0; i < qe.min; i++ {
		nextPositions := make(map[int]int)
		for _, p := range positions {
			nstr := str[p:]
			ok, lens := qe.expr.TestMatch(nstr)
			if ok {
				for _, r := range lens {
					nextPositions[p+r] = p+r
				}
			}
		}
		if len(nextPositions) == 0 {
			return false, []int{}
		}
		positions = nextPositions
		nextPositions = make(map[int]int)
	}
	allPositions := make(map[int]int)
	for _, p := range positions {
		allPositions[p] = p
	}
	for i := qe.min; qe.max < 0 || qe.max > i; i++ {
		nextPositions := make(map[int]int)
		for _, p := range positions {
			nstr := str[p:]
			ok, lens := qe.expr.TestMatch(nstr)
			if ok {
				for _, r := range lens {
					np := r+p
					if _, has := allPositions[np]; !has {
						allPositions[np] = np
						positions[np] = np
					}
				}
			}
		}
		if len(nextPositions) == 0 {
			break
		}
		positions = nextPositions
		nextPositions = make(map[int]int)
	} 
	if len(allPositions) == 0 {
		return false, []int{}
	}
	res := make([]int, 0, len(allPositions))
	for _, p := range allPositions {
		res = append(res, p)
	}
	sort.Ints(res)
	return true, res
}

func (qe *quantifiedExpression) GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int) {
	nextId := firstId
	s := newExpressionNdfaNode(nextId)
	nextId++
	r := newExpressionNdfaNode(nextId)
	nextId++
	s.initial = true
	r.accepting = true
	qexpr, ok := qe.expr.(NdfaNodeGenerator)
	if !ok {
		panic("quantified subexpression type "+reflect.TypeOf(qe.expr).String()+" does not receive NdfaNodeGenerator")
	}
	inits, accLen := qexpr.GenerateNdfaNodes(nextId)
	nextId = inits[len(inits)-1].Id()+1
	init, ok := inits[0].(*expressionNdfaNode)
	if !ok {
		panic("quantified subexpression node was not an *expressionNdfaNode")
	}
	init.initial = false
	work := make([]NdfaNode, len(inits))
	for k, n := range inits {
		work[k] = n
	}
	s.epsilons = append(s.epsilons, init)
	if qe.min <= 0 {
		s.epsilons = append(s.epsilons, r)
	} else {
		for i := 0; i < qe.min; i++ {
			next, accNext := qexpr.GenerateNdfaNodes(nextId)
			nextId = next[len(next)-1].Id()+1
			nextInit, ok := next[0].(*expressionNdfaNode)
			if !ok {
				panic("quantified subexpression node was not an *expressionNdfaNode")
			}
			init = nextInit
			for k := len(work)-accLen; k < len(work); k++ {
				acc, ok := work[k].(*expressionNdfaNode)
				if !ok {
					panic("quantified subexpression node was not an *expressionNdfaNode")
				}
				acc.accepting = false
				acc.epsilons = append(acc.epsilons, nextInit)
			}
			work = append(work, next...)
			accLen = accNext
		}
	}
	if qe.max < 0 || qe.max == math.MaxInt32 {
		for k := len(work)-accLen; k < len(work); k++ {
			acc, ok := work[k].(*expressionNdfaNode)
			if !ok {
				panic("quantified subexpression node was not an *expressionNdfaNode")
			}
			acc.accepting = false
			acc.epsilons = append(acc.epsilons, r)
			acc.epsilons = append(acc.epsilons, init)
		}
		work = append(work, r)
		return work, 1
	} else {
		if qe.min == qe.max {
			for k := len(work)-accLen; k < len(work); k++ {
				acc, ok := work[k].(*expressionNdfaNode)
				if !ok {
					panic("quantified subexpression node was not an *expressionNdfaNode")
				}
				acc.accepting = false
				acc.epsilons = append(acc.epsilons, r)
				work = append(work, r)
				return work, 1
			}
		}
		for i := 0; i < qe.max-qe.min; i++ {
			next, accNext := qexpr.GenerateNdfaNodes(nextId)
			nextId = next[len(next)-1].Id()+1
			nextInit, ok := next[0].(*expressionNdfaNode)
			if !ok {
				panic("quantified subexpression node was not an *expressionNdfaNode")
			}
			for k := len(work)-accLen; k < len(work); k++ {
				acc, ok := work[k].(*expressionNdfaNode)
				if !ok {
					panic("quantified subexpression node was not an *expressionNdfaNode")
				}
				acc.accepting = false
				acc.epsilons = append(acc.epsilons, r)
				acc.epsilons = append(acc.epsilons, nextInit)
				work = append(work, next...)
				accLen = accNext
			}
		}
	}
	work = append(work, r)
	return work, 1
}

func (nn *expressionNdfaNode) Id() uint32 {
	return nn.id
}

func (nn *expressionNdfaNode) Literals() []rune {
	res := make([]rune, 0, len(nn.literals))
	for c, _ := range nn.literals {
		res = append(res, c)
	}
	return res
}

func (nn *expressionNdfaNode) LiteralTransitions(c rune) []NdfaNode {
	if m, has := nn.literals[c]; has {
		res := make([]NdfaNode, len(m))
		for i, node := range m {
			res[i] = node
		}
		return res
	} else {
		return []NdfaNode{}
	}
}

func (nn *expressionNdfaNode) CharacterRanges() []CharacterRange {
	res := make([]CharacterRange, 0, len(nn.ranges))
	for r, _ := range nn.ranges {
		res = append(res, characterRange{rune(r & math.MaxInt32), rune(r >> 32)})
	}
	return res
}

func (nn *expressionNdfaNode) CharacterRangeTransitions(cr CharacterRange) []NdfaNode {
	hv := cr.Hash()
	if m, has := nn.ranges[hv]; has {
		res := make([]NdfaNode, len(m))
		for i, node := range m {
			res[i] = node
		}
		return res
	} else {
		return []NdfaNode{}
	}
}

func (nn *expressionNdfaNode) EpsilonTransitions() []NdfaNode {
	res := make([]NdfaNode, len(nn.epsilons))
	for i, node := range nn.epsilons {
		res[i] = node
	}
	return res
}

func (nn *expressionNdfaNode) IsTerminal() bool {
	return nn.accepting
}

func (nn *expressionNdfaNode) Query(c rune) []NdfaNode {
	var res []NdfaNode
	res = append(res, nn.LiteralTransitions(c)...)
	for _, r := range nn.CharacterRanges() {
		if r.Test(c) {
			res = append(res, nn.CharacterRangeTransitions(r)...)
		}
	}
	return res
}

func (nn *expressionNdfaNode) Domain() Domain {
	if nn.completed {
		return nn.domain
	}
	return nil
}

func (nn *expressionNdfaNode) Block() Block {
	if nn.completed {
		return nn.block
	}
	return nil
}

func (nn *expressionNdfaNode) Termdef() Termdef {
	if nn.completed {
		return nn.termdef
	}
	return nil
}

func (nn *expressionNdfaNode) IsIgnore() bool {
	return nn.ignore
}

func invertRegularizedIntervals(ranges []*characterRange) []*characterRange {
	if len(ranges) == 0 {
		return []*characterRange{&characterRange{0, -1}}
	}
	var lidx int
	var res []*characterRange
	r := &characterRange{}
	if ranges[0].least <= 0 {
		r.least = ranges[0].greatest + 1
		lidx = 1
	} else {
		r.least = 0
		r.greatest = ranges[0].least - 1
		res = append(res, r)
		r = &characterRange{}
		if ranges[0].greatest < 0 {
			return res
		}
		r.least = ranges[0].greatest + 1
		lidx = 1
	}
	for lidx < len(ranges) {
		r.greatest = ranges[lidx].least - 1
		if r.least < 0 {
			r.least = 0
		}
		res = append(res, r)
		r = &characterRange{}
		if ranges[lidx].greatest < 0 {
			return res
		}
		r.least = ranges[lidx].greatest + 1
		lidx++
	}
	r.greatest = -1
	if r.least < 0 {
		r.least = 0
	}
	res = append(res, r)
	return res
}

func matchEscapeCharacterLiteral(c rune, inCharset bool) string {
	switch(c) {
		case '$': return "\\$"
		case '^': return "\\^"
		case '*': return "\\*"
		case '+': return "\\+"
		case '?': return "\\?"
		case '|': return "\\|"
		case '{': return "\\{"
		case '}': return "\\}"
		case '[': return "\\["
		case ']': return "\\]"
		case '(': return "\\("
		case ')': return "\\)"
		case '.': return "\\."
		case '/': return "\\/"
		case '\\': return "\\\\"
		case '\n': return "\\n"
		case '\t': return "\\t"
		case '\r': return "\\r"
		case '\f': return "\\f"
		case 0: return "\\0"
	}
	if !unicode.IsPrint(c) {
		return "\\x"+strconv.Itoa(int(c))
	}
	return string([]rune{c})
}

func WriteExpression(e Expression, out io.Writer) {
	switch(e.Type()) {
		case MatchNever: {
			out.Write([]byte("\\_"))
		}
		case MatchAlways: {
			out.Write([]byte{'.'})
		}
		case MatchCharacterLiteral: {
			cl := e.(*characterLiteralExpression)
			out.Write([]byte(matchEscapeCharacterLiteral(cl.literal, false)))
		}
		case MatchStart: {
			out.Write([]byte{'^'})
		}
		case LexlMatchEnd: {
			out.Write([]byte{'$'})
		}
		case MatchSubmatch: {
			panic("submatch expressions unimplemented")
		}
		case MatchOptional: {
			panic("optional expressions unimplemented")
		}
		case MatchStar: {
			// XXX - Somewhere we should check that there is never a serial (ie.
			// sequence or alternation) expression as the argument to a suffixed
			// (star, plus, optional, quantified) one - that breaks operator 
			// precedence.  They must be grouped into a submatch.
			se := e.(*starExpression)
			WriteExpression(se.expr, out)
			out.Write([]byte{'*'})
		}
		case MatchPlus: {
			pe := e.(*plusExpression)
			WriteExpression(pe.expr, out)
			out.Write([]byte{'+'})
		}
		case MatchQuantified: {
			pq := e.(*quantifiedExpression)
			WriteExpression(pq.expr, out)
			if pq.min == pq.max {
				out.Write([]byte(fmt.Sprintf("{%d}", pq.max)))
			} else {
				if pq.min < 0 {
					out.Write([]byte(fmt.Sprintf("{,%d}", pq.max)))
				} else if pq.max < 0 {
					out.Write([]byte(fmt.Sprintf("{%d,}", pq.min)))
				} else {
					out.Write([]byte(fmt.Sprintf("{%d,%d}", pq.min, pq.max)))
				}
			}
		}
		case MatchCharset: {
			cse := e.(*characterClassExpression)
			if cse.class.Negated() {
				out.Write([]byte("[^"))
			} else {
				out.Write([]byte{'['})
			}
			for _, c := range cse.class.Literals() {
				out.Write([]byte(matchEscapeCharacterLiteral(c,true)))
			}
			for _, r := range cse.class.Ranges() {
				if r.Greatest() < 0 || r.Greatest() == math.MaxInt32 {
						out.Write([]byte(fmt.Sprintf("%s-", matchEscapeCharacterLiteral(r.Least(), true))))
				} else {
					out.Write([]byte(fmt.Sprintf("%s-%s", matchEscapeCharacterLiteral(r.Least(), true),
					                                      matchEscapeCharacterLiteral(r.Greatest(), true))))
				}
			}
			out.Write([]byte{']'})
		}
		case MatchSequence: {
			se := e.(*sequenceExpression)
			for _, expr := range se.exprs {
				WriteExpression(expr, out)
			}
		}
		case MatchAlternation: {
			se := e.(*alternationExpression)
			for i, expr := range se.exprs {
				WriteExpression(expr, out)
				if i < len(se.exprs)-1 {
					out.Write([]byte{'|'})
				}
			}
		}
		default: {
			panic("unknown expression type")
		}
	}
}