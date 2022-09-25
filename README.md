### Quoi Programming Language

[Pronunciation](https://forvo.com/word/quoi/) (like 'kwa')

Quoi is a simple programming language. This repository is an implementation of this language that compiles Quoi to Go.

Quoi is an explicitly, and statically typed programming language.

##### Some code samples

```

```

```
fun factorial(int n) -> int {
    ; factorial here
}
```

Quoi does not have a lot of features, or syntactic sugar.

There are 3 primitive data types: 
```
int         ; 64-bit
bool
string      ; utf-8 encoded strings
```

##### Operators
- Operators in Quoi are prefix operators (like in Lisp).
- They are enclosed in parenthesis ("like in Lisp" 2).
```
+ - * / @lt @lte @gt @gte
and or not
```
```
(* (+ 1 2) (/ 6 2))         ; result is 9
(and true true)             ; true
(not (and true false))      ; true 
(@lt 5 4)                   ; false
(not (@gte 5 5))            ; false
```

- There are lists.

**LISTS HERE**

<a id="datatypes"></a>
There are user-defined data types (```datatype```).

```
; declaration
datatype User {
    string name
    int age
}

; initialization
```
**INITIALIZATION**

See [Zero values](#zero-values)

##### Zero values

- "" for strings
- 0 for ints
- false for bools

Functions: 

``` 
; declaration
fun function_name([type arg,...]) -> return type[,return type2,...] {
    ...
    return .
}

; no return values
fun hello_world() {
    
}

fun greet(string name) {
    
}

fun some_func(int a, b) -> string, bool {
}
```

Loops:

```
loop <condition> {

}
```
As long as condition is true, call statements inside the block ({}).

```c
for (int i = 0; i < 10; i++) {
    printf("#%d\n", i);
}
```
Equivalent of this classical loop above: 
```

```

Branching:
```
if <condition> {

} elseif <condition> {

} else {

}
```

See [datatype](#datatypes)   

##### Keywords

List of all keywords: 

``` 
datatype, fun, int, string, bool, block, end, if, elseif, else, loop, return, and, or, not, lt, lte, gt, gte
```

--- 
##### Some notes about the syntax

- Statements end with dots.
- Spacing is not strict. As long as you separate keywords with at least one whitespace character, the rest doesn't matter.
- Escape sequences: 
  - \           to escape any character (e.g. "C:\\\\" is "C:\\"; "\\"" is escaping a quote, etc.)
  - \n          line feed
  - \r          carriage return
  - \t          horizontal tab
- Newlines are required after every field in ```datatype``` declarations.

##### Some notes about the semantics

- Quoi is a procedural language.
- It is explicitly, and statically typed.
- It does not allow function overloading.
- Functions can only be declared globally. No function declarations in other functions' bodies, or in any other block (ifs, loops, arbitrary blocks, etc.). 
- No way to make a variable constant, but there is a convention that ```ALL_UPPERCASE``` variables are meant to be constants (like in Python).
- You can create new blocks that have their own scopes, using ```block```, and ```end``` keywords.
- Variables in a scope, cannot be accessed outside of said scope. It will raise some kind of a ```ReferenceError``` (like in Javascript).
  - ```
    int day = 15.
    block 
        int day = 30.
        int age = 15.
        print day.      ; prints 30
                        ; if a global variable and a variable in a scope has the same name,
                        ; and a (pseudo-)function references that name, then the function will
                        ; use the one which is in the same block as it is. 
    end
    print day.          ; prints 15
    print age.          ; reference error
    ```
- Ability to compose different types to create a compound data type, using the ```datatype``` keyword.
- There are no methods attached to a data type, but you can just create functions that take in any data type.
  - ```
    datatype City {
        string name
        int founded_in
    }

    fun introduce_city(City c) -> string {
        string res = "City".
        return res.
    }
    ```
- No module system; but there are namespaces that you can use. They form the standard library.
  - When you use a namespace, the code necessary to provide that service is injected in the compiled Go code.
- No floats.
- Only one looping construct (```loop``` keyword).
- No manual memory management. Quoi programs are compiled to Go, and the Go runtime handles all the memory management using a garbage collector.
- All the code is written in one file (this may change). There is no entry point to the program (a main function), so the instructions just run sequentially, top to bottom. Compiled Go code is in one file.
- Global variables can be accessed anywhere in the program.
- No pointers, but values are pass-by-reference; meaning when you pass an argument to a function, you basically pass a pointer to that argument, so the callee can change the argument's value.
  - ```
    int age = 30.

    fun celebrate_birthday(int age) {
        ; increment age here (with something like ++ in other languages)
    }

    celebrate_birthday(age).    ; ...
    age                 ; 31
    ```
- No function signatures.
- Functions are not values. They cannot be assigned to variables.
- We can reference functions before their declarations.

##### Namespaces

- Namespaces form the standard library.
- They provide functions to manipulate built-in data types, or print to the console, etc.

Syntax:

```
<namespace>::<function>().
```

```
; get the index of the first occurence of character 'e' in string "Hello"
int idx = String::index("Hello", "e").
Stdout::print("Index of 'e': ").
Stdout::println(idx).
```