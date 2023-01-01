package analyzer

import (
	"fmt"
	"os"
	"quoi/lexer"
	"quoi/parser"
	"testing"
)

func _new(input string) *Analyzer {
	l := lexer.New(input)
	if len(l.Errs) > 0 {
		for _, v := range l.Errs {
			fmt.Printf("lexer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		os.Exit(1)
	}
	p := parser.New(l)
	program := p.Parse()
	if len(p.Errs) > 0 {
		for _, v := range p.Errs {
			fmt.Printf("parser err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		os.Exit(1)
	}
	a := New(program)
	return a
}

func TestFirstPass1(t *testing.T) {
	input := `
		fun hello(int a) -> int, string, bool, User {
			return 5.
		}
	`
	a := _new(input)
	a.Analyze()
	x := a.env.GetFunc("hello")
	fmt.Println(x)
	for _, v := range a.Errs {
		t.Logf("analyzer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
}

func TestFirstPass2(t *testing.T) {
	input := `
		datatype User { 
			string name
			int age
			City city
		}
	`
	a := _new(input)
	a.Analyze()
	x := a.env.GetDatatype("User")
	fmt.Println(x)
	for _, v := range a.Errs {
		t.Logf("analyzer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
}

func TestPrefExpr1(t *testing.T) {
	input := `
		int a = (+ 1 2).
		int a = "3".
		string a = (+ 1 2).
		int a = (+ "hello " "world").
		int a = (+ 1 "hello").
		int b = 5.
		bool a = (+ 1 b).
		`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestList1(t *testing.T) {
	input := `
			;listof int nx = ["hey", 2, 3].
			;listof string nx = ["hey", "2", "3"].
			;listof string names = ["jennifer"].
			;listof int numbers = [40, 50, 7, 567, 517].
			;listof int numbers2 = numbers.
			;listof int nx1 = 1.
			;listof int nx2 = [1, 2].
			;listof string strings = "hey".
			;listof string strings2 = [].
			;listof int a = [1].
			int a = 5.
			int b = a.
			;int c = (+ "hey" b).
			
			;listof int c = [(+ a b)].
			;listof string strings3 = c.

			;listof int nx = 5.
			listof int nx = [5].
			`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestOps1(t *testing.T) {
	input := `
		int a = (+ 1).
		int b = (+ "hey" " world").
		int c = (/ 2).
		int z = (lt 5 4).
		bool x = (not (lt 5 6)).
		bool q = (not (and true (lt 5 6)))
		`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestTopLevel1(t *testing.T) {
	input := `
		break.
		continue.
		(+ 1 2).
		`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestIf1(t *testing.T) {
	input := `
		string y = "Hello".
		if true {
			int x = 1.
		} elseif false {
			int y = 6.
		} else {
			; this 'y' should refer to 'string y = "Hello"' above.
			int x = y.
		}
		;if "hey" {}
		if (lt 5 6) {
			string q = "hey".
		}
		;bool p = (and true false).
		;string qq = q.
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}

func TestIf2(t *testing.T) {
	input := `
		if true {
			;datatype X {}
		} elseif false {
			fun w() -> {}
		} else {
		}
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}

func TestDatatype1(t *testing.T) {
	input := `
		datatype X {}
		;datatype X {}
		datatype Y {
			string x
			;int x
			int y
			User user	
		}

		;datatype User {}
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}

func TestOps2(t *testing.T) {
	input := `
		bool x = (= 5 5).
		;bool x = (= 5 "hey").
		;bool x = (= "hey" 5).
		;int x = (= "hey" "hey").
		;int x = (+ true true).

		int x = 555.
		int y, string q = x, "Hello".
		int total = (+ 1 2 3 q).
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}

func TestSubseq1(t *testing.T) {
	input := `
		listof int nx, listof string strx = [1, 2, 3], ["h", "e", "y"].
		listof int nx2, listof string strx2 = nx, strx.
		listof int nxq = strx.
		int x, listof string strx, bool y = 1, [], true.
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}

func TestReas1(t *testing.T) {
	input := `
		int x = 1.
		int y = x.
		x = 2.
		int q = x.
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestBlock1(t *testing.T) {
	input := `
		int x = 10.
		;int xx = 100.
		block 
			int x = 1.
			int y = (+ x 1).
			;string s = xx.
		end
		;int q = x.
		block 
;			break.
;			continue.
		end
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	//fmt.Println(program.IRStatements[1].(*IRBlock).Stmts[1].(*IRVariable).Value.(*IRPrefExpr).Operands[0].(*IRVariableReference).Value)
}

func TestLoop1(t *testing.T) {
	input := `
		;loop (+ 1 2) {
		;	int x = 1.
		;}

		int i = 0.
		loop (lt i 10) {
			i = (+ i 1).
			;fun a() -> string {}
			;datatype Song {
			;	string name
			;	int year
			;}
			break.
			continue.
		}
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestTopLevel2(t *testing.T) {
	input := `
		"Hello".
		1.
		true.
		User{}.
		(+ 1 2).
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestFun1(t *testing.T) {
	input := `
		fun a() -> {}
		fun b() -> string {}
		fun c() -> { return 1. }
		fun d() -> string { return 1. }
		fun e() -> string { return "Hello". }
		fun f(string b) -> string { return b. }
		int b = 6.
		fun g(string z) -> int { return b. }
		fun h(string b) -> string { return b. }
		fun j(listof string names) -> listof string, int { return names, 5. }
		fun CH(string b) -> int { return b. } 
		`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestFun2(t *testing.T) {
	/* 		fun a() -> int {
		if true {
			return 5.
		}
	}

			fun a() -> int {
			block end
			if true {}
			loop true {}
		}

				fun a() -> int {
			if true {
				if true {

				} elseif false {
					return "5".
				}
			}
		}
	*/
	input := `
		fun a() -> int, listof bool {
			loop true {
				if true {

				} elseif false {

				} else {
					return 5, [true, true].
					if true {
						if true {

						} else {
							block
								if true {
									return 1, [false, "hey"].
								} 
							end
						}
					}
				}
			}
		}
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestFailedVar1(t *testing.T) {
	input := `
		int x = "5".
		int y = x.
		string z = 4.

		listof int nx = (+ 1 1).
		listof string names = nx.
		listof bool m = [true, false].
		listof bool m = [true, false].
`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestFC1(t *testing.T) {
	input := `
	fun fn2() -> int, bool { return 5, true. }

	;int x, bool y, int z = fn2().
	int q, bool qb = fn2().
	bool yy = qb.
	int qq = yy.
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestGeneral1(t *testing.T) {
	input := `
		int my_age = 21.
		if (lte my_age 18) {
			;warn("you are not an adult").
		}
		string name, int age2 = "Jennifer", 44.
		int age_total = (+ my_age age2).

		datatype Person {
			string name
			int age
			bool alive
		}
		;Person j = Person { name=name age=age2 }.
		;Person j2 = Person { name=(+ 1 2) }.
		;Person j3 = Person { whatIsThis=Person{} }.
		
		fun example_function() -> string, bool { return "Hello", true. }

		;Person j4 = Person { name=example_function() }.

		listof string letters = ["A", "B", "C"].
		string B = (' letters 1).

		listof int numbers = [1, 2, 3, 4].
		int N = (' numbers 1).

		N = 1415.

		block 
			Person JENNIFER = Person { name="Jennifer" age=44 }. 
			Person p4 = JENNIFER.
		end
		;Person p4 = JENNIFER.
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}

	fmt.Println(program)
}

func TestGeneral2(t *testing.T) {
	input := `
	int my_age = 21.
	if (lte my_age 18) {
		;warn("you are not an adult").
	}
	string name, int age2 = "Jennifer", 44.
	int age_total = (+ my_age age2).

	datatype Person {
		string name
		int age
		bool alive
	}
	;Person j = Person { name=name age=age2 }.
	;Person j2 = Person { name=(+ 1 2) }.
	;Person j3 = Person { whatIsThis=Person{} }.
	
	fun example_function() -> string, bool { return "Hello", true. }

	;Person j4 = Person { name=example_function() }.

	listof string letters = ["A", "B", "C"].
	string B = (' letters 1).

	listof int numbers = [1, 2, 3, 4].
	int N = (' numbers 1).

	N = 1415.

	block 
		Person JENNIFER = Person { name="Jennifer" age=44 }. 
		Person p4 = JENNIFER.
	end
	;Person p4 = JENNIFER.

	fun give_me(string s) -> string {
		return s.
	}

	fun I_Return_Nothing() -> {}

	int s = give_me("hey").
	int vv = I_Return_Nothing().
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}

	fmt.Println(program)
}

func TestFC2(t *testing.T) {
	input := `
		;Stdout::println(1).
		;Stdout::println().
		;Stdout::println( String::concat( "#", String::from_int(5) ) ).
		
		;string x = Math::pow(1, 2).
		
		datatype City { 
			string name
		}

		;int x = Int::from_string( City { name="City 1" } ).
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}

	fmt.Println(program)
}

func TestFC3(t *testing.T) {
	input := `
		fun two_ints() -> int, int { return 5, 6. }

		;two_ints().
		;int n2, int n1 = two_ints().
		;int n3, string s1 = two_ints().
		;int n3, int n4, int n5 = two_ints().
		;int n6 = two_ints().

		fun takes_two_strs(string s1, string s2) -> {}
		fun returns_two_strs() -> string, string { return "Hello ", "world". }

		;takes_two_strs().
		takes_two_strs( returns_two_strs() ).

		fun takes_three_strs(string s1, string s2, string s3) -> {}

		takes_three_strs(returns_two_strs()).

		fun takes_nothing() -> {}

		takes_nothing( 1 ).

		fun takes_one_string(string s1) -> {}

		;takes_one_string(1).
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}

	fmt.Println(program)
}

func TestGetSet1(t *testing.T) {
	input := `	
		;int u = 5.
		datatype User {
			string name
			int age
		}

		User u = User {}.

		;int n = (get u name).
		;bool n = (get u name).
		string n = (get u name).
		;User user1 = (get u no_field).

		;u = (set u name 5).
		;u = (set u age "hey").
		;u = (set u unknown City{ name="City 1" }).
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}

	fmt.Println(program)
}

func TestGeneral3(t *testing.T) {
	input := `
		fun h() -> int {
			return 5.
		}
		fun factorial(int n) -> int {
			int product = 1.
			int j = 1.
			loop (lte j n) {
			j = (+ j 1).
			product = (* product j).
			}
			return product.
		}

		int i = 0.
		loop (lt i 10) {
			string msg = String::concat("#", String::from_int(i)).
			Stdout::println(msg).
			Stdout::print("\n").
			i = (+ i 1).
		}
		
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	fmt.Println(program)
}
