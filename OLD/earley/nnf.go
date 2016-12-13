package earley

import (
	//"fmt"
	//"strings"
	
	"errors"
	"github.com/dtromb/parser"
	"github.com/dtromb/parser/index"
	cnt "github.com/dtromb/collections/combinatorics"
)

func IsNihilistic(gp parser.GrammarParticle) (bool,error) {
	if gp.Epsilon() {
		return true, nil
	}
	idxg := parser.GetIndexedGrammar(gp.Grammar())
	idx, err := idxg.GetIndex(index.FIRST_FOLLOW_INDEX)
	if err != nil {
		return false, err
	}
	ffidx := idx.(*index.FFIndex)
	return ffidx.NumIns(gp) == 0, nil
}

func IsNihilisticNormalForm(g parser.Grammar) (bool,error) {
	idxg := parser.GetIndexedGrammar(g)
	idx, err := idxg.GetIndex(index.BASIC_INDEX)
	if err != nil {
		return false, err
	}
	bidx := idx.(*index.BasicGrammarIndex)
	for _, nt := range idxg.Nonterminals() {
		n, err := IsNihilistic(nt)
		if err != nil {
			return false, err
		}
		if bidx.Epsilon(nt) && !n {
			return false, nil
		}
	}
	return true, nil
}

func GetNihilisticAugmentGrammar(g parser.Grammar) (parser.Grammar, parser.SyntaxTreeTransform, error) {
	exceptionalEpsilons := []parser.GrammarParticle{}
	idxg := parser.GetIndexedGrammar(g)
	idx, err := idxg.GetIndex(index.BASIC_INDEX)
	if err != nil {
		return nil, nil, err
	}
	bidx := idx.(*index.BasicGrammarIndex)
	idx, err = idxg.GetIndex(index.NAME_INDEX)
	if err != nil {
		return nil, nil, err
	}
	nidx := idx.(*index.NameIndex)
	
	for _, nt := range idxg.Nonterminals() {
		n, err := IsNihilistic(nt)
		if err != nil {
			return nil, nil, err
		}
		if bidx.Epsilon(nt) && !n {
			exceptionalEpsilons = append(exceptionalEpsilons, nt)
		}
	}
	
	gb := parser.OpenGrammarBuilder()
	for _, nt := range g.Nonterminals() {
		if nt.Asterisk() {
			continue
		}
		gb.Nonterminals(nt.Name())
	}	
	for _, t := range g.Terminals() {
		if t.Epsilon() {
			continue
		}
		gb.Terminals(t.Name())
	}
	augmentMap := make(map[string]string)
	invMap := make(map[string]string)
	for _, e := range exceptionalEpsilons {
		eName := e.Name()+"-ε"
		augmentMap[e.Name()] = eName
		invMap[eName] = e.Name()
		gb.Nonterminals(eName)
		//gb.Rule().Lhs(eName).Rhs("`e")
	}
	for _, nt := range g.Nonterminals() {
		for i := 0; i < bidx.NumLhsStarts(nt); i++ {
			prod := bidx.LhsStart(nt,i)
			//fmt.Printf("LHSTART(%s,%d): %s\n", nt.String(), i, prod.String())
			exIdx := []int{}
			rhs := []string{}
			for j := 0; j < prod.RhsLen(); j++ {
				t := prod.Rhs(j)
				if _, has := augmentMap[t.Name()]; has {
					exIdx = append(exIdx,j)
					//fmt.Println("exidx gets "+t.String())
				}
				rhs = append(rhs, t.Name())
			}
			if nt.Asterisk() { // Initial rule is special-cased.
				//fmt.Println("rule transfers:  "+prod.String())
				//fmt.Println("Args: {"+prod.Lhs(0).Name()+"}, {"+strings.Join(rhs,",")+"}")
				gb.Rule().Lhs(prod.Lhs(0).Name()).Rhs(rhs...)
				initSym := nidx.Nonterminal(rhs[0])
				if initSym != nil {
					if bidx.Epsilon(initSym) {
						gb.Rule().Lhs(initSym.Name()).Rhs(augmentMap[initSym.Name()])
					}
				}
			} else {
				rhsinst := make([]string, len(rhs))
				//fmt.Printf("index len %d\n", len(exIdx))
				s := cnt.FirstSelection(uint(len(exIdx)))
				for {
					nnCount := 0
					for j := 0; j < len(rhs); j++ {
						rhsinst[j] = rhs[j]
						if nidx.Terminal(rhs[j]) != nil {
							nnCount++
						} else {
							nt := nidx.Nonterminal(rhs[j])
							nihil, err := IsNihilistic(nt)
							if err != nil {
								return nil, nil, err
							}
							if !nihil {
								nnCount++
							}
						}
					}
					copy(rhsinst, rhs)
					for j := 0; j < len(exIdx); j++ {
						if s.Test(j) {
							//fmt.Printf("idx %d replacement: %s->%s\n", exIdx[j], rhsinst[exIdx[j]], augmentMap[rhsinst[exIdx[j]]])
							rhsinst[exIdx[j]] = augmentMap[rhsinst[exIdx[j]]]
							nnCount--
						} 
					}
					var head string
					if nnCount == 0 {
						head = augmentMap[prod.Lhs(0).Name()]
					} else {
						head = prod.Lhs(0).Name()
					}
					//fmt.Println("rule transforms:  "+prod.String())
					//fmt.Println("Args: {"+head+"}, {"+strings.Join(rhs,",")+"}")
					gb.Rule().Lhs(head).Rhs(rhsinst...)
					if s.HasNext() {
						s = s.Next()
					} else {
						break
					}
				}
			}
		}
	}
	gb.Name(g.Name()+"-ε")
	augmentedGrammar, err := gb.Build()
	if err != nil {
		return nil, nil, err
	}
	augidx := parser.GetIndexedGrammar(augmentedGrammar)
	idx, err = augidx.GetIndex(index.NAME_INDEX)
	if err != nil {
		return nil, nil, err
	}
	anidx := idx.(*index.NameIndex)
	
	idx, err = augidx.GetIndex(index.BASIC_INDEX)
	if err != nil {
		return nil, nil, err
	}
	abidx := idx.(*index.BasicGrammarIndex)
	
	reverseMap := make(map[parser.GrammarParticle]parser.GrammarParticle)
	for k, v := range augmentMap {
		reverseMap[anidx.Nonterminal(v)] = nidx.Nonterminal(k)
	}
	prodMap := make(map[parser.Production]parser.Production)
	for _, p := range augmentedGrammar.Productions() {
		
		// Ignore the special case start rule for grammars that accept nil input.
		if p.LhsLen() == 1 && p.RhsLen() == 1 {
			initSym := abidx.LhsStart(augmentedGrammar.Asterisk(),0).Rhs(0)
			if p.Lhs(0) == initSym && invMap[p.Rhs(0).Name()] == initSym.Name() {
				continue
			}
		}
		if p.RhsLen() == 1 && p.Rhs(0).Epsilon() {
			continue
		}
		rhs := make([]string,0,p.RhsLen())
		for i := 0; i < p.RhsLen(); i++ {
			rp := p.Rhs(i)
			if ot, has := reverseMap[rp]; has {
				rhs = append(rhs, ot.Name())
			} else {
				rhs = append(rhs, rp.Name())
			}
		}
		//fmt.Printf("Searching for preimage rhs %s\n", strings.Join(rhs,","))
		var target parser.Production
		for _, cp := range nidx.RhsNames(rhs) {
			//fmt.Println("  considering "+cp.String())
			//for _, p := range g.Productions() {
				//fmt.Println(p.String())
			//}
			if nt, has := reverseMap[p.Lhs(0)]; has {
				if cp.Lhs(0) == nt {
					target = cp
					break
				}
			} else {
				if cp.Lhs(0).Name() == p.Lhs(0).Name() {
					target = cp
					break
				}
			}
		}
		if target == nil {
			return nil, nil, errors.New("Could not find preimage of augmented production rule: "+p.String())
		}
		prodMap[p] = target
	}
	var rtrans func(treeNode parser.SyntaxTreeNode)(parser.SyntaxTreeNode, error)
	
	rtrans = func(treeNode parser.SyntaxTreeNode)(parser.SyntaxTreeNode, error) {
		part := treeNode.Part()
		if op, has := reverseMap[part]; has {
			part = op
		}
		exp := make([]parser.SyntaxTreeNode, treeNode.NumChildren())
		for i := 0; i < len(exp); i++ {
			st := treeNode.Child(i)
			if _, has := reverseMap[st.Part()]; has {
				r, err := rtrans(st)
				if err != nil {
					return nil, err
				}
				exp[i] = r
			} else {
				exp[i] = st
			}
		}
		return &parser.BasicSyntaxTreeNode{
			Particle: part,
			FirstTokenIdx: treeNode.First(),
			LastTokenIdx: treeNode.Last(),
			SyntacticValue: treeNode.Value(),
			Prod: prodMap[treeNode.Rule()],
			Expansion: exp,
		}, nil
	}
	return augmentedGrammar, rtrans, nil
}