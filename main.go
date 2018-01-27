package main

import (
	"fmt"
	"glexer/glex"
	"strings"
)

func main() {
	sr := strings.NewReader(`name == "nombre"`)

	lexer := &glex.Lexer{}
	lexer.Init(sr)

	i := 0
	for tk := lexer.Scan(); tk != glex.EOF; tk = lexer.Scan() {
		fmt.Printf("[%d]: %v <%v>\n", i, glex.TokenNameFor(tk), lexer.TokenText())
		i++
	}

}
