package glex

// Reference link:
// 1. source code of golang std library of text/scanner
// 2. what is utf-8 ?
//    http://www.ruanyifeng.com/blog/2007/10/ascii_unicode_and_utf-8.html

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"text/scanner"
	"unicode"
	"unicode/utf8"
)

const (
	EOF          = -(iota + 1)
	Unknown      // as its name
	Var          // a,b
	ConstNumeric // 1,2,
	ConstString  // 'mundo'
	Equ          // ==
	And          // &&
	Or           // ||
	LeftBracket  // (
	RightBracket // )

	// Add          // +
	// Sub          // -
)

var tokenString = map[rune]string{
	EOF:          "EOF",
	Var:          "Var",
	ConstNumeric: "ConstantNumeric",
	ConstString:  "ConstString",
	Equ:          "EQU",
	And:          "AND",
	Or:           "OR",
	LeftBracket:  "LEFT-BRACKET",
	RightBracket: "RIGHT-BRACKET",
}

var (
	errHandlerNotImplemented = errors.New("Error not hanled")

	StringNotClosed = errors.New("strings not closed")
)

const bufLen = 1024

type Lexer struct {
	src    io.Reader
	srcBuf [bufLen + 1]byte
	srcPos int
	srcEnd int

	srcBufOffset int
	line         int
	column       int
	lastLineLen  int
	lastCharLen  int

	tokBuf bytes.Buffer
	tokPos int
	tokEnd int

	ch rune

	whitespace uint64
}

func (lx *Lexer) Init(src io.Reader) *Lexer {
	lx.src = src

	// Nothing has been in the buffer
	lx.srcBuf[0] = utf8.RuneSelf //
	lx.srcPos = 0
	lx.srcEnd = 0

	// initialize source position
	lx.srcBufOffset = 0
	lx.line = 1
	lx.column = 0
	lx.lastLineLen = 0
	lx.lastCharLen = 0

	lx.tokPos = -1

	lx.whitespace = scanner.GoWhitespace
	lx.ch = -2 // bytes lookahead
	return lx
}

func (lx *Lexer) error(msg string) {
	fmt.Fprintf(os.Stderr, "%v\n", msg)
}

func (lx *Lexer) next() rune {
	ch, width := rune(lx.srcBuf[lx.srcPos]), 1
	if ch >= utf8.RuneSelf {
		for lx.srcPos+utf8.UTFMax > lx.srcEnd &&
			!utf8.FullRune(lx.srcBuf[lx.srcPos:lx.srcEnd]) {
			if lx.tokPos >= 0 {
				lx.tokBuf.Write(lx.srcBuf[lx.tokPos:lx.srcPos])
				lx.tokPos = 0
			}
			copy(lx.srcBuf[0:], lx.srcBuf[lx.srcPos:lx.srcEnd])
			lx.srcBufOffset += lx.srcPos
			i := lx.srcEnd - lx.srcPos
			// n = 0 if not reading any bytes
			// with a non-nil err returned
			n, err := lx.src.Read(lx.srcBuf[i:bufLen])

			lx.srcPos = 0
			lx.srcEnd = i + n
			lx.srcBuf[lx.srcEnd] = utf8.RuneSelf
			if err != nil {
				if err != io.EOF {
					panic(errHandlerNotImplemented)
				}
				if lx.srcEnd == 0 {
					if lx.lastCharLen > 0 {
						lx.column++
					}
					lx.lastCharLen = 0
					return EOF
				}
				// rare. not getting more bytes. sad.
				break
			}
		}

		// width is still 1(bypassing utf8.RuneSelf)
		ch = rune(lx.srcBuf[lx.srcPos])
		if ch >= utf8.RuneSelf {
			ch, width = utf8.DecodeRune(lx.srcBuf[lx.srcPos:lx.srcEnd])
			// Differs from text.Scanner
			if ch == utf8.RuneError && width == 1 {
				lx.srcPos += width
				lx.lastCharLen = width
				lx.column++
				lx.error("illegal UTF-8 encoding!")
				return ch
			}
		}
	}

	// advance
	lx.srcPos += width
	lx.lastCharLen = width // last utf-8 char
	lx.column++
	switch ch {
	case 0:
		lx.error("illegal character NUL")
	case '\n':
		lx.line++
		lx.lastLineLen = lx.column
		lx.column = 0
	}
	return ch
}

func (lx *Lexer) peek() rune {
	if lx.ch == -2 {
		lx.ch = lx.next()
		// big endian utf-8
		if lx.ch == '\uFEFF' {
			lx.ch = lx.next()
		}
	}
	return lx.ch
}

// symbol name
func (lx *Lexer) Scan() rune {
	ch := lx.peek()

	//reset token text position
	lx.tokPos = -1

	for lx.whitespace&(1<<uint(ch)) != 0 {
		ch = lx.next()
	}

	// start collecting token text
	// srcPos is the position just past 'lookahead'
	lx.tokBuf.Reset()
	lx.tokPos = lx.srcPos - lx.lastCharLen

	tok := ch
	switch {
	case lx.isIdentRune(ch, 0):
		ch = lx.scanIdentifier()
		tok = Var
	case EOF == ch:
		break
	case '\'' == ch:
		fallthrough
	case '"' == ch:
		ch = lx.scanString(ch)
		tok = ConstString
	case '=' == ch:
		ch = lx.scanEqu()
		tok = Equ
	case '&' == ch:
		ch = lx.scanAnd()
		tok = And
	case '|' == ch:
		ch = lx.scanOr()
		tok = Or
	case '(' == ch:
		ch = lx.next()
		tok = LeftBracket
	case ')' == ch:
		ch = lx.next()
		tok = RightBracket
	case '0' <= ch && ch <= '9':
		ch = lx.scanNumber(ch)
		tok = ConstNumeric

	default:
		panic(fmt.Errorf("not implemented:<%v>", string(ch)))
	}

	lx.tokEnd = lx.srcPos - lx.lastCharLen
	lx.ch = ch
	return tok
}

func (lx *Lexer) scanIdentifier() rune {
	ch := lx.next()
	for i := 1; lx.isIdentRune(ch, i); i++ {
		ch = lx.next()
	}
	return ch
}

func (lx *Lexer) scanEqu() rune {
	ch := lx.next()
	if '=' == ch {
		ch = lx.next()
	} else {
		panic(fmt.Errorf("not ==! :%v", ch))
	}
	return ch
}

func (lx *Lexer) scanAnd() rune {
	ch := lx.next()
	if '&' == ch {
		ch = lx.next()
	} else {
		panic(fmt.Errorf("not &&! :%v", ch))
	}
	return ch
}

func (lx *Lexer) scanOr() rune {
	ch := lx.next()
	if '|' == ch {
		ch = lx.next()
	} else {
		panic(fmt.Errorf("not ||! :%v", ch))
	}
	return ch
}

func (lx *Lexer) scanString(right rune) rune {
	ch := lx.next()
	for ch != right && ch != EOF {
		ch = lx.next()
	}
	if right != ch {
		panic(StringNotClosed)
	}
	ch = lx.next() //skip right
	return ch
}

func (lx *Lexer) scanMantissa(ch rune) rune {
	for ch >= '0' && ch <= '9' {
		ch = lx.next()
	}
	return ch
}

func (lx *Lexer) scanNumber(ch rune) rune {
	ch = lx.scanMantissa(ch)
	if '.' == ch {
		ch = lx.scanMantissa(lx.next())
	}
	return ch
}

func (lx *Lexer) isIdentRune(ch rune, i int) bool {
	return '_' == ch ||
		unicode.IsLetter(ch) ||
		unicode.IsDigit(ch) && i > 0
}

func (lx *Lexer) TokenText() string {
	if lx.tokPos < 0 {
		return ""
	}
	if lx.tokEnd < 0 {
		lx.tokEnd = lx.tokPos
	}

	if lx.tokBuf.Len() == 0 {
		return string(lx.srcBuf[lx.tokPos:lx.tokEnd])
	}
	lx.tokBuf.Write(lx.srcBuf[lx.tokPos:lx.tokEnd])
	lx.tokPos = lx.tokEnd //idepotency
	return lx.tokBuf.String()
}

func TokenNameFor(tok rune) string {
	if name, ok := tokenString[rune(tok)]; ok {
		return name
	}
	return "*"
}
