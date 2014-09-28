
package libkbgo

import (
	"regexp"
	"fmt"
	"strings"
)

var empty_string []byte = []byte{}

const (
	NONE = iota
	OR = iota
	AND = iota
	LPAREN = iota
	RPAREN = iota
	URL = iota
	EOF = iota
	ERROR = iota
)

type Token struct {
	Typ int
	value []byte
}

func (t Token) getString() string {
	return string(t.value)
}

func byteArrayEq(a1, a2 []byte) bool {
	if len(a1) != len(a2) {
		return false;
	}
	for i,c := range(a1) {
		if c != a2[i] {
			return false;
		}	
	}
	return true;
}

func (t Token) Eq(t2 Token) bool {
	return (t.Typ == t2.Typ) && byteArrayEq(t.value, t2.value)	
}

func NewToken(typ int) (* Token) {
	return &Token { typ, empty_string }	
}

type Lexer struct {
	buffer []byte
	last *Token
	putback bool
	re *regexp.Regexp
	wss *regexp.Regexp
}

func NewLexer(s string) (* Lexer) {
	re := regexp.MustCompile(`^(\|\|)|(\&\&)|(\()|(\))|([^ \n\t&|()]+)`)
	wss := regexp.MustCompile(`^([\n\t ]+)`)
	l := &Lexer {[]byte(s), nil, false, re, wss};
	l.stripBuffer()
	return l;
}

func (lx *Lexer) stripBuffer() {
	if len(lx.buffer) > 0 {
		if match := lx.wss.FindSubmatchIndex(lx.buffer); match != nil {
			lx.buffer = lx.buffer[match[3]:]
		}
	}
}

func (lx *Lexer) advanceBuffer(i int) {
	lx.buffer = lx.buffer[i:]
	lx.stripBuffer()
}

func (lx *Lexer) Putback() {
	lx.putback = true
}

func (lx *Lexer) Get() (* Token) {
	var ret *Token
	if lx.putback {
		ret = lx.last
		lx.putback = false
	} else if len(lx.buffer) == 0 {
		ret = NewToken(EOF)
	} else if match := lx.re.FindSubmatchIndex(lx.buffer); match != nil {
		i := OR
		for ; i <= URL; i++ {
			if match[i*2] >= 0 {
				ret = &Token { i, lx.buffer[match[2*i]:match[2*i+1]] }
				lx.advanceBuffer(match[2*i+1])
				break
			}
		}	
	} else {
		lx.buffer = empty_string
		ret = NewToken(ERROR)
	}
	lx.last = ret
	return ret
}

type AssertionExpression interface {
	ToString() string
}

type AssertionOr struct {
	terms []AssertionExpression
}

func (a AssertionOr) ToString() string {
	v := make([]string, len(a.terms))
	for i,t := range a.terms {
		v[i] = t.ToString()
	}
	return fmt.Sprintf("(%s)", strings.Join(v, " || " ))
}

type AssertionAnd struct {
	factors []AssertionExpression
}

func (a AssertionAnd) ToString() string {
	v := make([]string, len(a.factors))
	for i,f := range a.factors {
		v[i] = f.ToString()
	}
	return fmt.Sprintf("(%s)", strings.Join(v, " && " ))
}

type AssertionUrl struct {
	Value string
}

func (a AssertionUrl) ToString() string {
	return a.Value	
}

type Parser struct {
	lexer *Lexer
	err error
}

func NewParser(lexer *Lexer) *Parser {
	ret := &Parser { lexer, nil }
	return ret
}

func NewAssertionAnd(left,right AssertionExpression) AssertionAnd {
	factors := []AssertionExpression{ left, right }
	return AssertionAnd { factors }	
}

func NewAssertionOr(left,right AssertionExpression) AssertionOr {
	terms := []AssertionExpression{ left, right }
	return AssertionOr { terms }	
}

func (p *Parser) Parse() AssertionExpression {
	return p.parseExpr()
}

func (p *Parser) parseTerm() (ret AssertionExpression) {
	factor := p.parseFactor()
	tok := p.lexer.Get()
	if tok.Typ == AND {
		term := p.parseTerm()
		ret = NewAssertionAnd(factor, term)		
	} else {
		ret = factor
		p.lexer.Putback()
	}
	return ret;
}


func (p *Parser) parseFactor() (ret AssertionExpression) {
	tok := p.lexer.Get()
	switch tok.Typ {
	case URL:
		ret = AssertionUrl { tok.getString() }
	case LPAREN:
		ex := p.parseExpr()
		tok = p.lexer.Get()
		if tok.Typ == RPAREN {
			ret = ex
		} else {
			ret = nil
			p.err = fmt.Errorf("Unbalanced parentheses")
		}
	default:
		p.err = fmt.Errorf("Unexpected token: %s", tok.getString())
	}
	return ret
}

func (p *Parser) parseExpr() (ret AssertionExpression) {
	term := p.parseTerm()
	tok := p.lexer.Get()
	if tok.Typ == OR {
		ex := p.parseExpr()
		ret = NewAssertionOr(term, ex)
	} else {
		ret = term
		p.lexer.Putback()
	}
	return ret;
}

func Parse(s string) (AssertionExpression, error) {
	lexer := NewLexer(s)
	parser := Parser { lexer, nil }
	ret := parser.Parse()
	return ret, parser.err
}