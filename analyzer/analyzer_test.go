package analyzer

import (
	"quoi/lexer"
	"quoi/parser"
	"testing"
)

func _DO(t *testing.T, input string) {
	l := lexer.New(input)
	for _, v := range l.Errs {
		t.Logf("Lexer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		return
	}
	p := parser.New(l)
	program := p.Parse()
	for _, v := range p.Errs {
		t.Logf("Parser err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		return
	}
	a := NewAnalyzer(program)
	a.FirstPass()
	for _, v := range a.Errs {
		t.Logf("Analyzer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		return
	}
}

func TestFP1(t *testing.T) {
	input := `
		;fun a() -> int, bool {
		;	return true, true.
		;}

		;fun b() -> bool {
		;	;return 1.
		;}

		fun c() -> string, bool {
			if (lt 5 6) {
				if (= 1 1) { return "true", true. }
				return "hello", true.
			}
			block 
				if (= 5 5) {
					if (not false) {
						return "Hehe", "hehe", 1.
					}
					return 1.
				}
			end
		}
	`
	_DO(t, input)
}

func TestFP2(t *testing.T) {
	// missing return statement
	input := `
		;fun a() -> string {
		;
		;}

		;fun hello() -> int {
		;	PRINT("Hello").
		;}

		fun b() -> string {
			if (= 5 5) {
				block end
				if (not (lte 7 6)) {
					block 
						 if (gt 6 5) {
							;return "1".
							;return true.
							;return 1, 1.
						 }
					end
				}
			} elseif (not false) {

			} elseif (= 6 6) {

			} else {
				;return "string".
			}
		}
	`
	_DO(t, input)
}

func TestFP3(t *testing.T) {
	input := `
		fun a() -> int {
			return 1.
		}

		fun b() -> int, string {
			return 1, "hello".	
		}

		fun c() -> {
			int i = 0.
			loop true {
				if (= i 150) { break. }
				string msg = String::concat("#", String::from_int(i), ": Hello!").
				Stdout::println(msg).
				i = (+ i 1).
			}
		}
	`
	_DO(t, input)
}

func TestFP4(t *testing.T) {
	input := `
		fun function1()->int{
			PRINT(123).
		}
	`
	_DO(t, input)
}
