package vgcfg

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const end_symbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleCONFIG
	ruleVARIABLE
	ruleVARNAME
	ruleGROUP
	ruleSTRING
	ruleINTEGER
	ruleLIST
	ruleSPACE
	ruleCOMMENT
	ruleAction0
	rulePegText
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"CONFIG",
	"VARIABLE",
	"VARNAME",
	"GROUP",
	"STRING",
	"INTEGER",
	"LIST",
	"SPACE",
	"COMMENT",
	"Action0",
	"PegText",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next uint32, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (ast *node32) Print(buffer string) {
	ast.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next uint32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: uint32(t.begin), end: uint32(t.end), next: uint32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = uint32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, uint32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth uint32, index int) {
	t.tree[index] = token32{pegRule: rule, begin: uint32(begin), end: uint32(end), next: uint32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/*func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2 * len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}*/

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type parser struct {
	baseParser

	Buffer string
	buffer []rune
	rules  [19]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range []rune(buffer) {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p *parser
}

func (e *parseError) Error() string {
	tokens, error := e.p.tokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ e.p.Buffer[begin:end] /*)*/)
	}

	return error
}

func (p *parser) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *parser) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *parser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.AddVariable()
		case ruleAction1:
			p.SetVarName(buffer[begin:end])
		case ruleAction2:
			p.BeginGroup()
		case ruleAction3:
			p.EndGroup()
		case ruleAction4:
			p.AddStringValue(buffer[begin:end])
		case ruleAction5:
			p.AddIntegerValue(buffer[begin:end])
		case ruleAction6:
			p.SetIsList(true)
		case ruleAction7:
			p.PrintComment(buffer[begin:end])

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *parser) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != end_symbol {
		p.buffer = append(p.buffer, end_symbol)
	}

	var tree tokenTree = &tokens32{tree: make([]token32, math.MaxInt16)}
	position, depth, tokenIndex, buffer, _rules := uint32(0), uint32(0), 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin uint32) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != end_symbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 CONFIG <- <(VARIABLE / ((&('#') COMMENT) | (&('\t' | '\n' | ' ') SPACE) | (&('+' | '-' | '.' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') GROUP)))+> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				{
					position4, tokenIndex4, depth4 := position, tokenIndex, depth
					if !_rules[ruleVARIABLE]() {
						goto l5
					}
					goto l4
				l5:
					position, tokenIndex, depth = position4, tokenIndex4, depth4
					{
						switch buffer[position] {
						case '#':
							if !_rules[ruleCOMMENT]() {
								goto l0
							}
							break
						case '\t', '\n', ' ':
							if !_rules[ruleSPACE]() {
								goto l0
							}
							break
						default:
							if !_rules[ruleGROUP]() {
								goto l0
							}
							break
						}
					}

				}
			l4:
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					{
						position7, tokenIndex7, depth7 := position, tokenIndex, depth
						if !_rules[ruleVARIABLE]() {
							goto l8
						}
						goto l7
					l8:
						position, tokenIndex, depth = position7, tokenIndex7, depth7
						{
							switch buffer[position] {
							case '#':
								if !_rules[ruleCOMMENT]() {
									goto l3
								}
								break
							case '\t', '\n', ' ':
								if !_rules[ruleSPACE]() {
									goto l3
								}
								break
							default:
								if !_rules[ruleGROUP]() {
									goto l3
								}
								break
							}
						}

					}
				l7:
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
				}
				depth--
				add(ruleCONFIG, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 VARIABLE <- <(VARNAME (' ' '=' ' ') ((&('[') LIST) | (&('"') STRING) | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') INTEGER)) Action0)> */
		func() bool {
			position10, tokenIndex10, depth10 := position, tokenIndex, depth
			{
				position11 := position
				depth++
				if !_rules[ruleVARNAME]() {
					goto l10
				}
				if buffer[position] != rune(' ') {
					goto l10
				}
				position++
				if buffer[position] != rune('=') {
					goto l10
				}
				position++
				if buffer[position] != rune(' ') {
					goto l10
				}
				position++
				{
					switch buffer[position] {
					case '[':
						{
							position13 := position
							depth++
							{
								position14, tokenIndex14, depth14 := position, tokenIndex, depth
								if buffer[position] != rune('[') {
									goto l15
								}
								position++
							l16:
								{
									position17, tokenIndex17, depth17 := position, tokenIndex, depth
									if !_rules[ruleSPACE]() {
										goto l17
									}
									goto l16
								l17:
									position, tokenIndex, depth = position17, tokenIndex17, depth17
								}
								if buffer[position] != rune(']') {
									goto l15
								}
								position++
								goto l14
							l15:
								position, tokenIndex, depth = position14, tokenIndex14, depth14
								if buffer[position] != rune('[') {
									goto l10
								}
								position++
								{
									position18, tokenIndex18, depth18 := position, tokenIndex, depth
									if !_rules[ruleSPACE]() {
										goto l18
									}
									goto l19
								l18:
									position, tokenIndex, depth = position18, tokenIndex18, depth18
								}
							l19:
								{
									position20, tokenIndex20, depth20 := position, tokenIndex, depth
									if !_rules[ruleSTRING]() {
										goto l21
									}
									goto l20
								l21:
									position, tokenIndex, depth = position20, tokenIndex20, depth20
									if !_rules[ruleINTEGER]() {
										goto l10
									}
								}
							l20:
							l22:
								{
									position23, tokenIndex23, depth23 := position, tokenIndex, depth
									{
										position24, tokenIndex24, depth24 := position, tokenIndex, depth
										if buffer[position] != rune(',') {
											goto l25
										}
										position++
										if buffer[position] != rune(' ') {
											goto l25
										}
										position++
										goto l24
									l25:
										position, tokenIndex, depth = position24, tokenIndex24, depth24
										{
											position26, tokenIndex26, depth26 := position, tokenIndex, depth
											if !_rules[ruleSTRING]() {
												goto l27
											}
											goto l26
										l27:
											position, tokenIndex, depth = position26, tokenIndex26, depth26
											if !_rules[ruleINTEGER]() {
												goto l23
											}
										}
									l26:
									}
								l24:
									goto l22
								l23:
									position, tokenIndex, depth = position23, tokenIndex23, depth23
								}
								{
									position28, tokenIndex28, depth28 := position, tokenIndex, depth
									if !_rules[ruleSPACE]() {
										goto l28
									}
									goto l29
								l28:
									position, tokenIndex, depth = position28, tokenIndex28, depth28
								}
							l29:
								if buffer[position] != rune(']') {
									goto l10
								}
								position++
							}
						l14:
							{
								add(ruleAction6, position)
							}
							depth--
							add(ruleLIST, position13)
						}
						break
					case '"':
						if !_rules[ruleSTRING]() {
							goto l10
						}
						break
					default:
						if !_rules[ruleINTEGER]() {
							goto l10
						}
						break
					}
				}

				{
					add(ruleAction0, position)
				}
				depth--
				add(ruleVARIABLE, position11)
			}
			return true
		l10:
			position, tokenIndex, depth = position10, tokenIndex10, depth10
			return false
		},
		/* 2 VARNAME <- <(<((&('-') '-') | (&('+') '+') | (&('.') '.') | (&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> Action1)> */
		func() bool {
			position32, tokenIndex32, depth32 := position, tokenIndex, depth
			{
				position33 := position
				depth++
				{
					position34 := position
					depth++
					{
						switch buffer[position] {
						case '-':
							if buffer[position] != rune('-') {
								goto l32
							}
							position++
							break
						case '+':
							if buffer[position] != rune('+') {
								goto l32
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l32
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l32
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l32
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l32
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l32
							}
							position++
							break
						}
					}

				l35:
					{
						position36, tokenIndex36, depth36 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '-':
								if buffer[position] != rune('-') {
									goto l36
								}
								position++
								break
							case '+':
								if buffer[position] != rune('+') {
									goto l36
								}
								position++
								break
							case '.':
								if buffer[position] != rune('.') {
									goto l36
								}
								position++
								break
							case '_':
								if buffer[position] != rune('_') {
									goto l36
								}
								position++
								break
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l36
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l36
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l36
								}
								position++
								break
							}
						}

						goto l35
					l36:
						position, tokenIndex, depth = position36, tokenIndex36, depth36
					}
					depth--
					add(rulePegText, position34)
				}
				{
					add(ruleAction1, position)
				}
				depth--
				add(ruleVARNAME, position33)
			}
			return true
		l32:
			position, tokenIndex, depth = position32, tokenIndex32, depth32
			return false
		},
		/* 3 GROUP <- <(VARNAME SPACE '{' Action2 (VARIABLE / ((&('#') COMMENT) | (&('\t' | '\n' | ' ') SPACE) | (&('+' | '-' | '.' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') GROUP)))+ '}' Action3)> */
		func() bool {
			position40, tokenIndex40, depth40 := position, tokenIndex, depth
			{
				position41 := position
				depth++
				if !_rules[ruleVARNAME]() {
					goto l40
				}
				if !_rules[ruleSPACE]() {
					goto l40
				}
				if buffer[position] != rune('{') {
					goto l40
				}
				position++
				{
					add(ruleAction2, position)
				}
				{
					position45, tokenIndex45, depth45 := position, tokenIndex, depth
					if !_rules[ruleVARIABLE]() {
						goto l46
					}
					goto l45
				l46:
					position, tokenIndex, depth = position45, tokenIndex45, depth45
					{
						switch buffer[position] {
						case '#':
							if !_rules[ruleCOMMENT]() {
								goto l40
							}
							break
						case '\t', '\n', ' ':
							if !_rules[ruleSPACE]() {
								goto l40
							}
							break
						default:
							if !_rules[ruleGROUP]() {
								goto l40
							}
							break
						}
					}

				}
			l45:
			l43:
				{
					position44, tokenIndex44, depth44 := position, tokenIndex, depth
					{
						position48, tokenIndex48, depth48 := position, tokenIndex, depth
						if !_rules[ruleVARIABLE]() {
							goto l49
						}
						goto l48
					l49:
						position, tokenIndex, depth = position48, tokenIndex48, depth48
						{
							switch buffer[position] {
							case '#':
								if !_rules[ruleCOMMENT]() {
									goto l44
								}
								break
							case '\t', '\n', ' ':
								if !_rules[ruleSPACE]() {
									goto l44
								}
								break
							default:
								if !_rules[ruleGROUP]() {
									goto l44
								}
								break
							}
						}

					}
				l48:
					goto l43
				l44:
					position, tokenIndex, depth = position44, tokenIndex44, depth44
				}
				if buffer[position] != rune('}') {
					goto l40
				}
				position++
				{
					add(ruleAction3, position)
				}
				depth--
				add(ruleGROUP, position41)
			}
			return true
		l40:
			position, tokenIndex, depth = position40, tokenIndex40, depth40
			return false
		},
		/* 4 STRING <- <('"' <(!((&('\r') '\r') | (&('\n') '\n') | (&('\\') '\\') | (&('"') '"')) .)*> '"' Action4)> */
		func() bool {
			position52, tokenIndex52, depth52 := position, tokenIndex, depth
			{
				position53 := position
				depth++
				if buffer[position] != rune('"') {
					goto l52
				}
				position++
				{
					position54 := position
					depth++
				l55:
					{
						position56, tokenIndex56, depth56 := position, tokenIndex, depth
						{
							position57, tokenIndex57, depth57 := position, tokenIndex, depth
							{
								switch buffer[position] {
								case '\r':
									if buffer[position] != rune('\r') {
										goto l57
									}
									position++
									break
								case '\n':
									if buffer[position] != rune('\n') {
										goto l57
									}
									position++
									break
								case '\\':
									if buffer[position] != rune('\\') {
										goto l57
									}
									position++
									break
								default:
									if buffer[position] != rune('"') {
										goto l57
									}
									position++
									break
								}
							}

							goto l56
						l57:
							position, tokenIndex, depth = position57, tokenIndex57, depth57
						}
						if !matchDot() {
							goto l56
						}
						goto l55
					l56:
						position, tokenIndex, depth = position56, tokenIndex56, depth56
					}
					depth--
					add(rulePegText, position54)
				}
				if buffer[position] != rune('"') {
					goto l52
				}
				position++
				{
					add(ruleAction4, position)
				}
				depth--
				add(ruleSTRING, position53)
			}
			return true
		l52:
			position, tokenIndex, depth = position52, tokenIndex52, depth52
			return false
		},
		/* 5 INTEGER <- <(<[0-9]+> Action5)> */
		func() bool {
			position60, tokenIndex60, depth60 := position, tokenIndex, depth
			{
				position61 := position
				depth++
				{
					position62 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l60
					}
					position++
				l63:
					{
						position64, tokenIndex64, depth64 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l64
						}
						position++
						goto l63
					l64:
						position, tokenIndex, depth = position64, tokenIndex64, depth64
					}
					depth--
					add(rulePegText, position62)
				}
				{
					add(ruleAction5, position)
				}
				depth--
				add(ruleINTEGER, position61)
			}
			return true
		l60:
			position, tokenIndex, depth = position60, tokenIndex60, depth60
			return false
		},
		/* 6 LIST <- <((('[' SPACE* ']') / ('[' SPACE? (STRING / INTEGER) ((',' ' ') / (STRING / INTEGER))* SPACE? ']')) Action6)> */
		nil,
		/* 7 SPACE <- <((&('\n') '\n') | (&('\t') '\t') | (&(' ') ' '))+> */
		func() bool {
			position67, tokenIndex67, depth67 := position, tokenIndex, depth
			{
				position68 := position
				depth++
				{
					switch buffer[position] {
					case '\n':
						if buffer[position] != rune('\n') {
							goto l67
						}
						position++
						break
					case '\t':
						if buffer[position] != rune('\t') {
							goto l67
						}
						position++
						break
					default:
						if buffer[position] != rune(' ') {
							goto l67
						}
						position++
						break
					}
				}

			l69:
				{
					position70, tokenIndex70, depth70 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '\n':
							if buffer[position] != rune('\n') {
								goto l70
							}
							position++
							break
						case '\t':
							if buffer[position] != rune('\t') {
								goto l70
							}
							position++
							break
						default:
							if buffer[position] != rune(' ') {
								goto l70
							}
							position++
							break
						}
					}

					goto l69
				l70:
					position, tokenIndex, depth = position70, tokenIndex70, depth70
				}
				depth--
				add(ruleSPACE, position68)
			}
			return true
		l67:
			position, tokenIndex, depth = position67, tokenIndex67, depth67
			return false
		},
		/* 8 COMMENT <- <('#' <(!('\r' / '\n') .)*> ('\r' / '\n')+ Action7)> */
		func() bool {
			position73, tokenIndex73, depth73 := position, tokenIndex, depth
			{
				position74 := position
				depth++
				if buffer[position] != rune('#') {
					goto l73
				}
				position++
				{
					position75 := position
					depth++
				l76:
					{
						position77, tokenIndex77, depth77 := position, tokenIndex, depth
						{
							position78, tokenIndex78, depth78 := position, tokenIndex, depth
							{
								position79, tokenIndex79, depth79 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l80
								}
								position++
								goto l79
							l80:
								position, tokenIndex, depth = position79, tokenIndex79, depth79
								if buffer[position] != rune('\n') {
									goto l78
								}
								position++
							}
						l79:
							goto l77
						l78:
							position, tokenIndex, depth = position78, tokenIndex78, depth78
						}
						if !matchDot() {
							goto l77
						}
						goto l76
					l77:
						position, tokenIndex, depth = position77, tokenIndex77, depth77
					}
					depth--
					add(rulePegText, position75)
				}
				{
					position83, tokenIndex83, depth83 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l84
					}
					position++
					goto l83
				l84:
					position, tokenIndex, depth = position83, tokenIndex83, depth83
					if buffer[position] != rune('\n') {
						goto l73
					}
					position++
				}
			l83:
			l81:
				{
					position82, tokenIndex82, depth82 := position, tokenIndex, depth
					{
						position85, tokenIndex85, depth85 := position, tokenIndex, depth
						if buffer[position] != rune('\r') {
							goto l86
						}
						position++
						goto l85
					l86:
						position, tokenIndex, depth = position85, tokenIndex85, depth85
						if buffer[position] != rune('\n') {
							goto l82
						}
						position++
					}
				l85:
					goto l81
				l82:
					position, tokenIndex, depth = position82, tokenIndex82, depth82
				}
				{
					add(ruleAction7, position)
				}
				depth--
				add(ruleCOMMENT, position74)
			}
			return true
		l73:
			position, tokenIndex, depth = position73, tokenIndex73, depth73
			return false
		},
		/* 10 Action0 <- <{ p.AddVariable() }> */
		nil,
		nil,
		/* 12 Action1 <- <{ p.SetVarName(buffer[begin:end]) }> */
		nil,
		/* 13 Action2 <- <{ p.BeginGroup() }> */
		nil,
		/* 14 Action3 <- <{ p.EndGroup() }> */
		nil,
		/* 15 Action4 <- <{ p.AddStringValue(buffer[begin:end]) }> */
		nil,
		/* 16 Action5 <- <{ p.AddIntegerValue(buffer[begin:end]) }> */
		nil,
		/* 17 Action6 <- <{ p.SetIsList(true) }> */
		nil,
		/* 18 Action7 <- <{ p.PrintComment(buffer[begin:end]) }> */
		nil,
	}
	p.rules = _rules
}
