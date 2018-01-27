package glex

import (
	"fmt"
	"strings"
	"testing"
)

func TestBigCase0(t *testing.T) {
	var lexer Lexer
	lexer.Init(strings.NewReader(`
        2.9 1234567890
        model == "strike" || model == "dump"
        20 == age && age == 28.30
        name == nombre
        12 3.145
        `))
	i := 0
	for t := lexer.Scan(); t != EOF; t = lexer.Scan() {
		fmt.Printf("[%d]: %v <%v>\n", i, TokenNameFor(t), lexer.TokenText())
		i++
	}
}

func TestBigCase1(t *testing.T) {
	sr := strings.NewReader(`
        1.3 3.141592653
       `)

	lexer := &Lexer{}
	lexer.Init(sr)

	i := 0
	for tk := lexer.Scan(); tk != EOF; tk = lexer.Scan() {
		fmt.Printf("[%d]: %v <%v>\n", i, TokenNameFor(tk), lexer.TokenText())
		i++
	}
}

func TestBigCase2(t *testing.T) {
	sr := strings.NewReader(`
        width == 2 || width == 4.3
       `)

	var lexer Lexer
	lexer.Init(sr)
	for i := 0; i < 1000; i++ {
		lexer.Scan()
	}
	// for i := 0; i < 1000; i++ {
	// 	lexer.next()
	// }
}
