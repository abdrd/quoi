### Quoi Programming Language

[Pronunciation](https://forvo.com/word/quoi/) (like 'kwa')

Quoi is a simple programming language. This repository is an implementation of this language that compiles Quoi to Go.

Quoi is an explicitly, and statically typed programming language.

##### Some code samples
```
; statements end with a dot (.).
print "Hello world".
```

```
fun factorial(int n) -> int {
    int res = 0.
    loop @gte n 1 {
        res = @mul res n.
        n = @sub n 1.
    }
    return res
}
```

Quoi does not have a lot of features, or syntactic sugar.

There are 3 primitive data types: 
```
int         ; 64-bit
bool
string      ; utf-8 encoded strings
```

There are lists.

```
list int ux = @listnew int ().      ; can pass initialization values in ()s. parens are required, even if
                                    ; there are not any values passed initially.
; append to list
@listpush ux 1.

; get element with index 
; int first_el = @listget ux 1.
@listget ux 1.                      ; get second element (indices start from zero).

; replace element with index
@listreplace ux 1 10.               ; make second element's value 10

; delete element with index
@listdelete ux 1.
```
<a id="datatypes"></a>
There are user-defined data types (```datatype```).

```
; declaration
datatype User {
    string name
    int age
}

; initialization

; we do not have to pass values for every field of User here. they are set to zero values by default.
; No args: 
;   User u = @new User ().
```
See [Zero values](#zero-values)
```
User u = @new User (name: "John", age: 61).

; getters

string name = @get u name.

; setters

@set u name "Johnny". 
```

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
    print "Hello world".
}

fun greet(string name) {
    print @strconcat "Hello " name.
}

fun some_func(int a, b) -> string, bool {
    int n = @add a b.
    ; this is a bit unreadable.
    ; string ret1 = @strconcat @str n @str a.
    ; let's do this instead:

    string strN = @str n.
    string strA = @str a.
    string ret1 = @strconcat strN strA.
    bool ret2 = true.
    return ret1, ret2.
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
int i = 0.
loop @lt i 10 { 
    ; can't pass more than one argument to print statement
    print @strconcat "#" @str i.
    @inc i          ; or i = @add i 1
}
```

Branching:
```
if <condition> {

} elseif <condition> {

} else {

}
```
```
int n = 10.

if @gt n 100 {

} elseif @eq n 11 {

} else {

}
```

##### Built-in functions (statements/pseudo-functions)
 
I call them pseudo-functions, because they are not called like user-defined functions are called (syntactically).

They usually start with the "at" symbol (@). Some of them produce values, while some of them do not.

Here's the list:
```
print                   ; take one (1) string to print to standard output.
printf                  ; take one (1) formatted string, and arguments needed. No newlines at the end.
                            ; format specifiers: 
                            ; %s        for strings
                            ; %d        for integers
                            ; %b        for bools

; these take in numbers, and return an integer result.
@add a b                ; add a, and b
@sub a b                ; subtract a from b 
@div a b                ; divide a, by b 
@mul a b                ; multiply a with b
@inc a                  ; increment a by 1
@dec a                  ; decrement a by 1
@str a                  ; convert integer a, to a string

; comparison functions used for integers:
; these produce boolean values.
@gt a b                 ; return a > b
@lt a b                 ; return a < b
@gte a b                ; return a >= b
@lte a b                ; return a <= b
@eq a b                 ; return whether a is equal to b
@neq a b                ; the opposite of @eq

; logical operators
@and a b                ; logical and
@or a b                 ; logical or
@not a                  ; negate a boolean value

; string functions
; return strings
@strget s idx           ; get the character at index idx in s
@strdelete s idx        ; delete ^^^
@strreplace s idx c     ; replace ^^^, with c
@strindex s c           ; return the index of the first occurence of character c, in s
@strconcat s s2         ; concatenate s, and s2
@streq s s2

; list functions

@listnew type (args)                    ; very similar to @new. see lists section
@listpush list value                    ; append to list
@listget list idx                       ; get element with index idx in list
@listreplace list idx new_value         ; replace element with index
@listdelete ux 1                        ; delete element with index
```
```
; datatype functions

@new type (args)
@get type field
@set type field value
```
See [datatype](#datatypes)   

##### Keywords

List of all keywords: 

``` 
print, printf, datatype, fun, int, string, bool, block, end, if, elseif, else, loop, return
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
        string res = @strconcat @get c name @strconcat " was founded in " @get c founded_in.
        return res.
    }
    ```
- No modules, packages, or a standard library.
- No floats.
- Only one looping construct (```loop``` keyword).
- No manual memory management. Quoi programs are compiled to Go, and the Go runtime handles all the memory management using a garbage collector.
- All the code is written in one file (this may change). There is no entry point to the program (a main function), so the instructions just run sequentially, top to bottom. Compiled Go code is in one file.
- Global variables can be accessed anywhere in the program.
- No pointers, but values are pass-by-reference; meaning when you pass an argument to a function, you basically pass a pointer to that argument, so the callee can change the argument's value.
  - ```
    int age = 30.

    fun celebrate_birthday(int age) {
        printf "\t\tHappy birthday\n".
        print @strconcat 
                    @strconcat 
                        "You are now "
                        @str age 
                    " years old.".
        @inc age.
    }

    celebrate_birthday(age).    ; ...
    print age.                  ; 31
    ```
- No function signatures.
- Functions are not values. They cannot be assigned to variables.
- We can reference functions before their declarations.