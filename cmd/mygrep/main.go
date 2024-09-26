package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Ensures gofmt doesn't remove the "bytes" import above (feel free to remove this!)
var _ = bytes.ContainsAny

// Usage: echo <input_text> | your_program.sh -E <pattern>
func main2() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	pattern := os.Args[2]

	line, err := io.ReadAll(os.Stdin) // assume we're only dealing with a single line
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	switch {
	case pattern == "\\d":
		ok := matchDigits(line)
		if !ok {
			os.Exit(1)
		}
	case pattern == "\\w":
		ok := matchAlphaNumeric(line)
		if !ok {
			os.Exit(1)
		}
	case len(pattern) > 3 && pattern[0] == '[' && pattern[len(pattern)-1] == ']' && pattern[1] == '^':
		ok := matchnNegativeCharGroups(line, pattern[2:len(pattern)-1])
		if !ok {
			os.Exit(1)
		}
	case len(pattern) > 2 && pattern[0] == '[' && pattern[len(pattern)-1] == ']':
		ok := matchPositiveCharGroups(line, pattern[1:len(pattern)-1])
		if !ok {
			os.Exit(1)
		}
	default:
		ok, err := matchLine(line, pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		if !ok {
			os.Exit(1)
		}

	}

	// default exit code is 0 which means success
}

func matchLine(line []byte, pattern string) (bool, error) {
	if utf8.RuneCountInString(pattern) != 1 {
		return false, fmt.Errorf("unsupported pattern: %q", pattern)
	}

	var ok bool

	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this to pass the first stage
	ok = bytes.ContainsAny(line, pattern)

	return ok, nil
}

func matchDigits(line []byte) bool {
	ok := bytes.ContainsAny(line, "012345678")
	return ok
}

func matchAlphaNumeric(line []byte) bool {
	pattern := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	ok := bytes.ContainsAny(line, pattern)
	return ok

}

func matchPositiveCharGroups(line []byte, pattern string) bool {
	ok := bytes.ContainsAny(line, pattern)
	return ok
}

func matchnNegativeCharGroups(line []byte, pattern string) bool {
	for _, c := range line {
		if !bytes.ContainsAny([]byte(pattern), string(c)) {
			return true
		}
	}
	return false
}

type matcher interface {
	match(r rune) bool
	isLiteral() bool
}

type digitMatcher struct{}

func (d digitMatcher) match(r rune) bool {
	return unicode.IsDigit(r)
}

func (d digitMatcher) isLiteral() bool {
	return false
}

type wordMatcher struct{}

func (w wordMatcher) match(r rune) bool {
	return unicode.IsDigit(r) || unicode.IsLetter(r) || r == '-'
}

func (w wordMatcher) isLiteral() bool {
	return false
}

type literalMatcher struct {
	char rune
}

func (l literalMatcher) match(r rune) bool {
	return l.char == r
}

func (l literalMatcher) isLiteral() bool {
	return true
}

type negativeCharGroupMatcher struct {
	chars string
}

func (n negativeCharGroupMatcher) match(r rune) bool {
	return !strings.ContainsRune(n.chars, r)
}

func (n negativeCharGroupMatcher) isLiteral() bool {
	return false
}

type positiveCharGroupMatcher struct {
	chars string
}

func (n positiveCharGroupMatcher) match(r rune) bool {
	return strings.ContainsRune(n.chars, r)
}

func (n positiveCharGroupMatcher) isLiteral() bool {
	return false
}

func parsePattern(pattern string) []matcher {
	var matchers []matcher
	for i := 0; i < len(pattern); i++ {

		if pattern[i] == '\\' && i + 1 < len(pattern) {
			switch pattern[i+1] {
			case 'd':
				matchers = append(matchers, digitMatcher{})
			case 'w':
				matchers = append(matchers, wordMatcher{})	
			default:
				matchers = append(matchers, literalMatcher{char: rune(pattern[i+1])})
			}
		} else if pattern[i] == '[' && i + 1 < len(pattern) {
			end := strings.IndexByte(pattern[i:], ']')

			if end == -1 {
				matchers = append(matchers, literalMatcher{char: rune(pattern[i])})
			} else {
				if pattern[i+1] == '^' {
					matchers = append(matchers, negativeCharGroupMatcher{chars: pattern[i+2:i+end]})
				} else {
					// postive char group
					matchers = append(matchers, positiveCharGroupMatcher{chars: pattern[i+1:i+end]})
				}
				i += end	
			}
		} else {
			matchers = append(matchers, literalMatcher{char: rune(pattern[i])})
		}
	}
	return matchers
}


func matchPattern(input string, pattern string) bool {
	matchers := parsePattern(pattern)
	
	for startIdx := 0; startIdx < len(input); startIdx++ {
		inputIdx, matcherIdx := startIdx, 0

		for inputIdx < len(input) && matcherIdx < len(matchers) {
			if matchers[matcherIdx].match(rune(input[inputIdx])) {
				inputIdx++
				matcherIdx++
			} else if matcherIdx > 0 && !matchers[matcherIdx-1].isLiteral() {
				// If the previous matcher was non-literal (like \d or \w), we can try the next matcher
				matcherIdx++
			} else {
				// If we don't match, break and try starting from the next character
				break
			}
		}

		if matcherIdx == len(matchers) {
			return true // We found a complete match
		}
	}

	return false // No match found in the entire input
}


func main() {
	if len(os.Args) != 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "Usage: %s -E <pattern>\n", os.Args[0])
		os.Exit(2)
	}

	pattern := os.Args[2]
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(2)
	}

	if matchPattern(string(input), pattern) {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
