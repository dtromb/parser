package lexr

import (
	"sort"
	"errors"
	"math"
	"github.com/dtromb/parser"
)

type DomainBuilder interface {
	Block(name string) DomainBuilder
	Termdef(termName string, expr Expression) DomainBuilder
	Include(blockName string) DomainBuilder
	Ignore(expr Expression) DomainBuilder
	ToBlock(blockName string) DomainBuilder
	DefaultToBlock(blockName string) DomainBuilder
	Build() (Domain, error)
	MustBuild() Domain
}

type CharacterClassBuilder interface {
	Negate() CharacterClassBuilder
	AddCharacter(rune) CharacterClassBuilder
	AddRange(least, greatest rune) CharacterClassBuilder
	Build() (CharacterClass, error)
	MustBuild() CharacterClass
}

func OpenDomainBuilder(grammar parser.Grammar) (DomainBuilder, error) {
	ig := parser.GetIndexedGrammar(grammar)
	tgIf, err := ig.GetIndex(parser.GrammarIndexTypeTerm)
	if err != nil {
		return nil, err
	}
	return &domainBuilder{
		grammar: ig,
		blockInfosByName: make(map[string]*stdDomainBlockInfo),
		currentTermdefInfosByName: make(map[string]*stdDomainTermdefInfo),
		termIndex: tgIf.(parser.TermGrammarIndex),
	}, nil
}

func OpenCharacterClassBuilder() CharacterClassBuilder {
	return &characterClassBuilder{
		literals: make(map[rune]bool),
		ranges: make([][]rune, 0, 4),
	}
}

///

type domainBuilder struct {
	grammar parser.Grammar
	termIndex parser.TermGrammarIndex
	hasBlock bool
	hasDefaultTo bool
	currentBlock string
	currentDefaultTo string
	currentIncludes []string
	currentIgnore Expression
	currentTermdefInfos []*stdDomainTermdefInfo
	currentTermdefInfosByName map[string]*stdDomainTermdefInfo
	blockInfos []*stdDomainBlockInfo
	blockInfosByName map[string]*stdDomainBlockInfo
}
	
type stdDomainBlockInfo struct {
	name string
	index int
	termdefs []*stdDomainTermdefInfo
	includeBlockNames []string
	ignore Expression
	defaultToBlockName string
}

type stdDomainTermdefInfo struct {
	name string
	nextBlock string
	hasNextBlock bool
	expr Expression
}

type stdDomain struct {
	grammar parser.Grammar
	termIndex parser.TermGrammarIndex
	blocks []*stdDomainBlock
}

type stdDomainBlock struct {
	domain *stdDomain
	name string
	index int
	termdefs []*stdDomainTermdef
	includeBlocks []*stdDomainBlock
	ignoreExpr Expression
	defaultNext *stdDomainBlock
}

type stdDomainTermdef struct {
	block *stdDomainBlock
	index int
	terminal parser.Term
	nextBlock *stdDomainBlock
	expr Expression
}

func (db *domainBuilder) finishBlock() error {
	if !db.hasBlock {
		return errors.New("Block() not called before finishing block")
	}
	bInfo := &stdDomainBlockInfo{
		name: db.currentBlock,
		index: len(db.blockInfos),
		includeBlockNames: db.currentIncludes,
		ignore: db.currentIgnore,
		termdefs: db.currentTermdefInfos,
		defaultToBlockName: db.currentDefaultTo,
	}
	db.blockInfos = append(db.blockInfos, bInfo)
	db.blockInfosByName[bInfo.name] = bInfo
	db.currentBlock = ""
	db.hasBlock = false
	db.currentDefaultTo = ""
	db.hasDefaultTo = false
	db.currentIgnore = nil
	db.currentIncludes = []string{}
	db.currentTermdefInfosByName = make(map[string]*stdDomainTermdefInfo)
	db.currentTermdefInfos = []*stdDomainTermdefInfo{}
	return nil
}

func (db *domainBuilder) Block(name string) DomainBuilder {
	if db.hasBlock {
		err := db.finishBlock()
		if err != nil {
			panic(err.Error())
		}
	}
	if _, has := db.blockInfosByName[name]; has {
		panic("duplicate block name")
	}
	db.currentBlock = name
	db.hasBlock = true
	return db
}

func (db *domainBuilder) Termdef(termName string, expr Expression) DomainBuilder {
	if _, has := db.currentTermdefInfosByName[termName]; has {
		panic("duplicate terminal definition")
	}
	if _, err := db.termIndex.GetTerminal(termName); err != nil {
		panic("grammar does not support named terminal")
	}
	ti := &stdDomainTermdefInfo{
		name: termName,
		expr: expr,
	}
	db.currentTermdefInfos = append(db.currentTermdefInfos, ti)
	db.currentTermdefInfosByName[termName] = ti
	return db
}

func (db *domainBuilder) ToBlock(blockName string) DomainBuilder {
	if len(db.currentTermdefInfos) == 0 {
		panic("ToBlock() called before Termdef()")
	}
	td := db.currentTermdefInfos[len(db.currentTermdefInfos)-1]
	if td.hasNextBlock {
		panic("ToBlock() called twice after same termdef")
	}
	td.hasNextBlock = true
	td.nextBlock = blockName
	return db
}

func (db *domainBuilder) Include(blockName string) DomainBuilder {
	if !db.hasBlock {
		panic("Include() called before Block()")
	}
	db.currentIncludes = append(db.currentIncludes, blockName)
	return db
}

func (db *domainBuilder) Ignore(expr Expression) DomainBuilder {
	if !db.hasBlock {
		panic("Ignore() called before Block()")
	}
	if db.currentIgnore != nil {
		panic("Ignore() called twice after same Block()")
	}
	db.currentIgnore = expr
	return db
}

func (db *domainBuilder) DefaultToBlock(blockName string) DomainBuilder {
	if !db.hasBlock {
		panic("DefaultToBlock() called before Block()")
	}
	if db.hasDefaultTo {
		panic("DefaultToBlock() called twice after same Block()")
	}
	db.hasDefaultTo = true
	db.currentDefaultTo = blockName
	return db
}

func (db *domainBuilder) Build() (Domain, error) {
	if db.hasBlock {
		err := db.finishBlock()
		if err != nil {
			return nil, err
		}
	}
	domain := &stdDomain{
		grammar: db.grammar, 
		termIndex: db.termIndex,
		blocks: make([]*stdDomainBlock, len(db.blockInfos)),
	}
	for i := 0; i < len(domain.blocks); i++ {
		blockInfo := db.blockInfos[i]
		domain.blocks[i] = &stdDomainBlock{
			domain: domain,
			name: blockInfo.name,
			index: i,
			termdefs: make([]*stdDomainTermdef,len(blockInfo.termdefs)),
			includeBlocks: make([]*stdDomainBlock,len(blockInfo.includeBlockNames)),
			ignoreExpr: blockInfo.ignore,
		}
	}
	
	for i := 0; i < len(domain.blocks); i++ {
		block := domain.blocks[i]
		blockInfo := db.blockInfos[i]
		for j := 0; j < len(block.termdefs); j++ {
			term, err := domain.termIndex.GetTerminal(blockInfo.termdefs[j].name)
			if err != nil {
				return nil, err
			}
			var nextBlock *stdDomainBlock
			if blockInfo.termdefs[j].nextBlock != "" {
				nextBlockInfo, has := db.blockInfosByName[blockInfo.termdefs[j].nextBlock]
				if !has {
					return nil, errors.New("forward to non-existant block '"+blockInfo.termdefs[j].nextBlock+"'")
				}
				nextBlock = domain.blocks[nextBlockInfo.index]
			}
			termdef := &stdDomainTermdef{
				block: block,
				index: j,
				terminal: term,
				nextBlock: nextBlock,
				expr: blockInfo.termdefs[j].expr,
			}
			block.termdefs[j] = termdef
		}
		for j := 0; j < len(block.includeBlocks); j++ {
			includeInfo, has := db.blockInfosByName[blockInfo.includeBlockNames[j]]
			if !has {
				return nil, errors.New("include of non-existant block '"+blockInfo.termdefs[j].nextBlock+"'")
			}
			block.includeBlocks[j] = domain.blocks[includeInfo.index]
		}
		if blockInfo.defaultToBlockName != "" {
			defaultTo, has := db.blockInfosByName[blockInfo.defaultToBlockName]
			if !has {
				return nil, errors.New("default forward to non-existant block '"+blockInfo.defaultToBlockName+"'")
			}
			block.defaultNext = domain.blocks[defaultTo.index]
		}
	}
	return domain, nil
}

func (db *domainBuilder) MustBuild() Domain {
	domain, err := db.Build()
	if err != nil {
		panic(err.Error())
	}	
	return domain
}

func cloneDomain(d Domain) (*stdDomain,error) {
	ig := parser.GetIndexedGrammar(d.Grammar())
	termIndexIf, err := ig.GetIndex(parser.GrammarIndexTypeTerm)
	if err != nil {
		return nil, err
	}
	sd := &stdDomain{
		grammar: ig,
		termIndex: termIndexIf.(parser.TermGrammarIndex),
		blocks: make([]*stdDomainBlock, d.NumBlocks()),
	}
	for i := 0; i < len(sd.blocks); i++ {
		block := d.Block(i)
		newBlock := &stdDomainBlock{
			domain: sd,
			name: block.Name(),
			index: i,
			termdefs: make([]*stdDomainTermdef, block.NumTermdefs()),
			includeBlocks: make([]*stdDomainBlock, block.NumInclusions()),
			ignoreExpr: block.Ignore(),
		}
		sd.blocks[i] = newBlock
	}
	for i := 0; i < len(sd.blocks); i++ {
		newBlock := sd.blocks[i]
		oldBlock := d.Block(i)
		for j := 0; j < len(newBlock.termdefs); j++ {
			oldTd := oldBlock.Termdef(j)
			t, _ := sd.termIndex.GetTerminal(oldTd.Terminal().Name())
			td := &stdDomainTermdef{
				block: newBlock,
				terminal: t,
				expr: oldTd.Expression(),
			}
			if oldTd.HasNextBlock() {
				td.nextBlock = sd.blocks[oldTd.NextBlock().Index()]
			}
			newBlock.termdefs[j] = td
		}
		for j := 0; j < len(newBlock.includeBlocks); j++ {
			newBlock.includeBlocks[j] = sd.blocks[oldBlock.Inclusion(j).Index()]
		}
		if oldBlock.HasDefaultForward() {
			newBlock.defaultNext = sd.blocks[oldBlock.DefaultForward().Index()]
		}
	}
	return sd, nil
}

	
func (d *stdDomain) NumBlocks() int {
	return len(d.blocks)
}

func (d *stdDomain) Block(idx int) Block {
	if idx < 0 || idx >= len(d.blocks) {
		panic("block index out of range")
	}
	return d.blocks[idx]
}

func (d *stdDomain) Grammar() parser.Grammar {
	return d.grammar
}

func (db *stdDomainBlock) Domain() Domain {
	return db.domain
}

func (db *stdDomainBlock) Name() string {
	return db.name
}

func (db *stdDomainBlock) Index() int {
	return db.index
}

func (db *stdDomainBlock) NumTermdefs() int {
	return len(db.termdefs)
}

func (db *stdDomainBlock) Termdef(idx int) Termdef {
	if idx < 0 || idx >= len(db.termdefs) {
		panic("termdef index out of bounds")
	}
	return db.termdefs[idx]
}

func (db *stdDomainBlock) NumInclusions() int {
	return len(db.includeBlocks)
}

func (db *stdDomainBlock) Inclusion(idx int) Block {
	if idx < 0 || idx >= len(db.includeBlocks) {
		panic("include block index out of bounds")
	}
	return db.includeBlocks[idx]
}

func (db *stdDomainBlock) Ignore() Expression {
	return db.ignoreExpr
}

func (db *stdDomainBlock) HasDefaultForward() bool {
	return db.defaultNext != nil
}

func (db *stdDomainBlock) DefaultForward() Block {
	if db.defaultNext == nil {
		return db
	}
	return db.defaultNext
}

func (dt *stdDomainTermdef) Terminal() parser.Term {
	return dt.terminal
}

func (dt *stdDomainTermdef) Expression() Expression {
	return dt.expr
}

func (dt *stdDomainTermdef) NextBlock() Block {
	if dt.nextBlock == nil {
		return dt.block
	}
	return dt.nextBlock
}

func (dt *stdDomainTermdef) Block() Block {
	return dt.block
}

func (dt *stdDomainTermdef) Index() int {
	return dt.index
}

func (dt *stdDomainTermdef) HasNextBlock() bool {
	return dt.nextBlock != nil
}

type characterClassBuilder struct {
	negated bool
	literals map[rune]bool
	ranges [][]rune
}

func (ccb *characterClassBuilder) Negate() CharacterClassBuilder {
	ccb.negated = !ccb.negated
	return ccb
}

func (ccb *characterClassBuilder) AddCharacter(c rune) CharacterClassBuilder {
	ccb.literals[c] = true
	return ccb
}

func (ccb *characterClassBuilder) AddRange(least, greatest rune) CharacterClassBuilder {
	if least > greatest && greatest > 0 {
		panic("least value in range may not be greater than greatest")
	}
	ccb.ranges = append(ccb.ranges, []rune{least,greatest})
	return ccb
}

func (ccb *characterClassBuilder) Build() (CharacterClass, error) {
	ranges := make([]*characterRange, 0, len(ccb.ranges) + len(ccb.literals))
	for _, r := range ccb.ranges {
		if r[0] < 0 {
			r[0] = rune(0)
		}
		if r[1] < 0 {
			r[1] = rune(math.MaxInt32)
		}
		ranges = append(ranges, &characterRange{r[0],r[1]})
	}
	for c, _ := range ccb.literals {
		ranges = append(ranges, &characterRange{c,c})
	}
	if len(ranges) == 0 {
		return nil, errors.New("empty character class, AddRange() nor AddLiteral() called before Build()")
	}
	cc := &stdCharacterClass{
		negated: ccb.negated,
		literals: make(map[rune]rune),
		ranges: make([]characterRange, 0, 4),
	}
	ranges = regularizeBoundedIntervals(ranges)
	for _, r := range ranges {
		if r.least == r.greatest {
			cc.literals[r.least] = r.least
		} else {
			if r.least == r.greatest - 1 {
				cc.literals[r.least] = r.least
				cc.literals[r.greatest] = r.greatest
			} else {
				if r.greatest == rune(math.MaxInt32) {
					r.greatest = -1
				}
				cc.ranges = append(cc.ranges, characterRange{r.least,r.greatest})
			}
		}
	}
	return cc, nil
} 

func (ccb *characterClassBuilder) MustBuild() CharacterClass {
	cc, err := ccb.Build()
	if err != nil {
		panic(err.Error())
	}
	return cc
}

func (cc *stdCharacterClass) Negated() bool {
	return cc.negated
}

func (cc *stdCharacterClass) Literals() []rune {
	res := make([]rune, 0, len(cc.literals))
	for _, c := range cc.literals {
		res = append(res, c)
	}
	return res
}

func (cc *stdCharacterClass) Ranges() []CharacterRange {
	res := make([]CharacterRange, 0, len(cc.ranges))
	for _, r := range cc.ranges {
		res = append(res, r)
	}
	return res
}

func (cr characterRange) Least() rune {
	return cr.least
}

func (cr characterRange) Greatest() rune {
	return cr.greatest
}

func (cr characterRange) Test(c rune) bool {
	return (cr.least < 0 || cr.least <= c) &&
	       (cr.greatest < 0 || cr.greatest >= c)
}

func (cr characterRange) Hash() uint64 {
	l := cr.Least()
	r := cr.Greatest()
	if l < 0 {
		l = 0
	}
	if r < 0 {
		r = math.MaxInt32
	}
	return (uint64(r) << 32) | uint64(l)
}

type leastSortedRanges []*characterRange
func (lsr leastSortedRanges) Len() int           { return len(lsr) }
func (lsr leastSortedRanges) Less(i, j int) bool { return lsr[i].least < lsr[j].least }
func (lsr leastSortedRanges) Swap(i, j int)      { lsr[i], lsr[j] = lsr[j], lsr[i] }

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