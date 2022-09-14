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
; See [Zero values][#zero-values]

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

} else if @eq n 11 {

} else {

}
```

##### Built-in functions (statements/pseudo-functions)
 
I call them pseudo-functions, because they are not called like user-defined functions are called (syntactically).

They usually start with the "at" symbol (@). Some of them produce values, while some of them do not.

Here's the list:
```
print                 ; take one (1) string to print to standard output.

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
@not a                  ; negate a boolean value
@neq a b                ; the opposite of @eq

; string functions
; return strings
@strget s idx           ; get the character at index idx in s
@strdelete s idx        ; delete ^^^
@strreplace s idx c     ; replace ^^^, with c
@strindex s c           ; return the index of the first occurence of character c, in s
@strconcat s s2         ; concatenate s, and s2

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