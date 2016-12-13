package lexl

import (
	"fmt"
	"unicode"
	//"fmt"
	"sort"
)

type CharacterClass interface {
	IsNegated() bool
	Literals() []rune
	Ranges() []CharacterRange
}

type CharacterRange interface {
	Least() rune
	Greatest() rune
}

type MutableCharacterClass interface {
	CharacterClass
	Negate()
	AddCharacter(rune) bool
	AddRange(least, greatest rune) bool
}

type CharacterClassExpression interface {
	MatchExpr
	CharacterClass() CharacterClass
}

//

type characterRange struct {
	least    rune
	greatest rune
}

func (cr characterRange) Least() rune {
	return cr.least
}

func (cr characterRange) Greatest() rune {
	return cr.greatest
}

type characterClass struct {
	negated  bool
	literals map[rune]rune
	ranges   []characterRange
}

func newCharacterClass() MutableCharacterClass {
	return &characterClass{
		literals: make(map[rune]rune),
	}
}

func (cc *characterClass) Literals() []rune {
	r := make([]rune, 0, len(cc.literals))
	for _, v := range cc.literals {
		r = append(r, v)
	}
	return r
}

func (cc *characterClass) Ranges() []CharacterRange {
	r := make([]CharacterRange, 0, len(cc.ranges))
	for _, v := range cc.ranges {
		r = append(r, v)
	}
	return r
}

func (cc *characterClass) Negate() {
	cc.negated = !cc.negated
}

func (cc *characterClass) IsNegated() bool {
	return cc.negated
}

func (cc *characterClass) AddCharacter(c rune) bool {
	if _, has := cc.literals[c]; has {
		return false
	}
	for _, r := range cc.ranges {
		if r.greatest >= c && r.least <= c {
			return false
		}
	}
	cc.literals[c] = c
	return true
}

func (cc *characterClass) AddRange(least, greatest rune) bool {
	for _, v := range cc.literals {
		if least <= v && greatest >= v {
			return false
		}
	}
	for _, r := range cc.ranges {
		if r.Greatest() >= least && greatest >= r.Least() {
			return false
		}
	}
	cc.ranges = append(cc.ranges, characterRange{least: least, greatest: greatest})
	return true
}

func CloneCharacterClass(cc CharacterClass) CharacterClass {
	ncc := newCharacterClass()
	if cc.IsNegated() {
		ncc.Negate()
	}
	for _, r := range cc.Ranges() {
		if !ncc.AddRange(r.Least(), r.Greatest()) {
			panic("malformed character class (range overlap)")
		}
	}
	for _, c := range cc.Literals() {
		if !ncc.AddCharacter(c) {
			panic("malformed character class (character overlap)")
		}
	}
	return ncc
}

func newLexlCharacterClassExpr(cclass CharacterClass) CharacterClassExpression {
	return &stdLexlCharacterClassExpression{
		cclass: CloneCharacterClass(cclass).(*characterClass),
	}
}

type stdLexlCharacterClassExpression struct {
	cclass *characterClass
}

func (*stdLexlCharacterClassExpression) Type() MatchExprType {
	return LexlMatchCharset
}

func (cce *stdLexlCharacterClassExpression) CharacterClass() CharacterClass {
	return cce.cclass
}

func newLexlWhitespaceClassExpr() CharacterClassExpression {
	cc := newCharacterClass()
	cc.AddCharacter(' ')
	cc.AddCharacter('\t')
	cc.AddCharacter('\n')
	cc.AddCharacter('\r')
	cc.AddCharacter('\f')
	return newLexlCharacterClassExpr(cc)
}

type leastSortedRanges []*characterRange
type greatestSortedRanges []*characterRange

func (lsr leastSortedRanges) Len() int           { return len(lsr) }
func (lsr leastSortedRanges) Less(i, j int) bool { return lsr[i].least < lsr[j].least }
func (lsr leastSortedRanges) Swap(i, j int)      { lsr[i], lsr[j] = lsr[j], lsr[i] }

func (gsr greatestSortedRanges) Len() int           { return len(gsr) }
func (gsr greatestSortedRanges) Less(i, j int) bool { return gsr[i].greatest < gsr[j].greatest }
func (gsr greatestSortedRanges) Swap(i, j int)      { gsr[i], gsr[j] = gsr[j], gsr[i] }

func invertRegularizedIntervals(ranges []*characterRange) []*characterRange {
	if len(ranges) == 0 {
		return []*characterRange{&characterRange{-1, -1}}
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

func regularizeBoundedIntervals(ranges []*characterRange) []*characterRange {
	lSort := leastSortedRanges(make([]*characterRange, len(ranges)))
	copy(lSort, ranges)
	sort.Sort(lSort)
	lidx := 0
	var res []*characterRange
	for lidx < len(lSort) {
		lRange := &characterRange{
			least:    lSort[lidx].least,
			greatest: lSort[lidx].greatest,
		}
		//fmt.Printf("%d: (%d,%d)\n", lidx, lRange.least, lRange.greatest)
		for lidx < len(lSort) && lSort[lidx].least-1 <= lRange.greatest {
			// The intervals overlap; merge them.
			//if lidx < len(lSort)-1 {
			//	fmt.Printf("merged, next is (%d,%d)\n", lSort[lidx+1].least, lSort[lidx+1].greatest)
			//} else {
			//	fmt.Printf("merged, at end of list\n")
			//}
			lRange.greatest = lSort[lidx].greatest
			lidx++
		}
		//fmt.Printf("done merging, printing (%d,%d)\n", lRange.least, lRange.greatest)
		res = append(res, lRange)
		if lRange.greatest < 0 {
			break
		}
	}
	return res
}

func (cc *stdLexlCharacterClassExpression) GenerateNdfaStates() (states []*stdLexlNdfaState, err error) {
	fmt.Println("CHARCLASS")
	cls := cc.cclass
	init := newStdLexlNdfaState()
	final := newStdLexlNdfaState()
	final.accepting = true
	if cls.negated {
		var ranges []*characterRange
		for _, c := range cls.literals {
			ranges = append(ranges, &characterRange{c, c})
		}
		for _, r := range cls.ranges {
			ranges = append(ranges, &characterRange{r.Least(), r.Greatest()})
		}
		ranges = invertRegularizedIntervals(regularizeBoundedIntervals(ranges))
		for _, r := range ranges {
			if r.Greatest() == r.Least() {
				init.literals[r.Least()] = []*stdLexlNdfaState{final}
			} else {
				init.ranges[r] = []*stdLexlNdfaState{final}
			}
		}
	} else {
		for _, c := range cls.literals {
			init.literals[c] = []*stdLexlNdfaState{final}
		}
		for _, r := range cls.ranges {
			init.ranges[r] = []*stdLexlNdfaState{final}
		}
	}
	return []*stdLexlNdfaState{init, final}, nil
}

func (cc *stdLexlCharacterClassExpression) isPrintableClassChar(c rune) bool {
	return unicode.IsPrint(c) && c <= 256
}

func (cc *stdLexlCharacterClassExpression) isSpecialClassChar(c rune) bool {
	if c > 255 {
		return false
	}
	switch byte(c) {
	case '^':
		fallthrough
	case '$':
		fallthrough
	case '\\':
		fallthrough
	case '-':
		fallthrough
	case ']':
		{
			return true
		}
	}
	return false
}

func (cc *stdLexlCharacterClassExpression) appendClassChar(buf []rune, c rune) []rune {
	if cc.isPrintableClassChar(c) {
		if cc.isSpecialClassChar(c) {
			buf = append(buf, '\\')
		}
		buf = append(buf, c)
	} else {
		switch c {
		case '\n':
			buf = append(buf, []rune("\\n")...)
		case '\t':
			buf = append(buf, []rune("\\t")...)
		case '\r':
			buf = append(buf, []rune("\\r")...)
		case '\f':
			buf = append(buf, []rune("\\f")...)
		default:
			{
				buf = append(buf, ([]rune)(fmt.Sprintf("\\x%4.4x", c))...)
			}
		}
	}
	return buf
}

func (cc *stdLexlCharacterClassExpression) ToString() string {
	var buf []rune
	buf = append(buf, '[')
	if cc.cclass.negated {
		buf = append(buf, '^')
	}
	for _, c := range cc.cclass.literals {
		buf = cc.appendClassChar(buf, c)
	}
	for _, r := range cc.cclass.ranges {
		buf = cc.appendClassChar(buf, r.least)
		buf = append(buf, '-')
		buf = cc.appendClassChar(buf, r.greatest)
	}
	buf = append(buf, ']')
	return string(buf)
}
