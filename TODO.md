### Lexer
- [ ] Fix column, and line reporting.

### Parser
- [ ] Context-aware error recovery
- [ ] Refactor
- [ ] Embed lexer instead of embedding a token stream for memory efficiency.
- [ ] List types in datatype fields

### Semantic analyzer
- [ ] Refactor `sema/symbol.go`

### Compiler 
- [ ] Show erroneous line-of-code in error messages. For example:
  ```
  Stdout::println( (lt 5 true) ).
                         ^
  10:24 TypeError: invalid expression of type 'bool' for 'lt' operator
  ```
