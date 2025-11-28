import re
from typing import List, Optional
from .tokens import Token, TokenType

class LexerError(Exception):
    def __init__(self, message, line, column):
        super().__init__(f"{message} at line {line}, column {column}")
        self.line = line
        self.column = column

class Lexer:
    def __init__(self, source: str):
        self.source = source
        self.pos = 0
        self.line = 1
        self.column = 1
        self.tokens: List[Token] = []
        self.indent_stack = [0]  # Stack of indentation levels (column numbers)
        
        self.keywords = {
            'def': TokenType.DEF,
            'async': TokenType.ASYNC,
            'return': TokenType.RETURN,
            'if': TokenType.IF,
            'else': TokenType.ELSE,
            'while': TokenType.WHILE,
            'for': TokenType.FOR,
            'in': TokenType.IN,
            'spawn': TokenType.SPAWN,
            'await': TokenType.AWAIT,
            'True': TokenType.TRUE,
            'False': TokenType.FALSE,
            'None': TokenType.NONE,
            'lambda': TokenType.LAMBDA,
        }

    def tokenize(self) -> List[Token]:
        while self.pos < len(self.source):
            char = self.peek()
            
            if char == '#':
                self.skip_comment()
            elif char == '\n':
                self.handle_newline()
            elif char.isspace():
                self.advance()
            elif char.isalpha() or char == '_':
                self.tokens.append(self.read_identifier())
            elif char.isdigit():
                self.tokens.append(self.read_number())
            elif char == '"' or char == "'":
                self.tokens.append(self.read_string())
            else:
                self.handle_symbol()
                
        self.handle_eof()
        return self.tokens

    def peek(self) -> str:
        if self.pos >= len(self.source):
            return ''
        return self.source[self.pos]
    
    def advance(self):
        if self.pos < len(self.source):
            if self.source[self.pos] == '\n':
                self.line += 1
                self.column = 1
            else:
                self.column += 1
            self.pos += 1

    def skip_comment(self):
        while self.peek() != '\n' and self.peek() != '':
            self.advance()

    def handle_newline(self):
        self.advance() # Consume \n
        
        # Check for empty lines or comment-only lines
        start_pos = self.pos
        while self.peek().isspace() and self.peek() != '\n' and self.peek() != '':
            self.advance()
            
        if self.peek() == '\n' or self.peek() == '#':
            return # Ignore empty lines
            
        if self.peek() == '':
            return # EOF after newline

        # Calculate indentation
        indent_level = 0
        # Reset to start of line to count spaces properly
        # Actually, since we advanced, column is reset to 1. 
        # But we need to count spaces from the start of the line.
        # Let's just count the spaces we skipped.
        # Wait, self.column is 1 after \n. 
        # The loop above advanced over spaces. 
        # So current column - 1 is the indentation level if we assume 1-based indexing and start at 1.
        indent_level = self.column - 1
        
        current_indent = self.indent_stack[-1]
        
        # Emit NEWLINE for the previous line
        self.tokens.append(Token(TokenType.NEWLINE, line=self.line-1, column=1))

        if indent_level > current_indent:
            self.indent_stack.append(indent_level)
            self.tokens.append(Token(TokenType.INDENT, line=self.line, column=1))
        elif indent_level < current_indent:
            while indent_level < self.indent_stack[-1]:
                self.indent_stack.pop()
                self.tokens.append(Token(TokenType.DEDENT, line=self.line, column=1))
            if indent_level != self.indent_stack[-1]:
                raise LexerError("Inconsistent indentation", self.line, 1)
        else:
            pass

    def read_identifier(self) -> Token:
        start_col = self.column
        value = ''
        while self.peek().isalnum() or self.peek() == '_':
            value += self.peek()
            self.advance()
            
        token_type = self.keywords.get(value, TokenType.IDENTIFIER)
        return Token(token_type, value, self.line, start_col)

    def read_number(self) -> Token:
        start_col = self.column
        value = ''
        is_float = False
        while self.peek().isdigit() or self.peek() == '.':
            if self.peek() == '.':
                if is_float: break
                is_float = True
            value += self.peek()
            self.advance()
            
        return Token(TokenType.NUMBER, float(value) if is_float else int(value), self.line, start_col)

    def read_string(self) -> Token:
        start_col = self.column
        quote = self.peek()
        self.advance()
        value = ''
        while self.peek() != quote and self.peek() != '':
            if self.peek() == '\\':
                self.advance()
                # Handle escapes if needed
            value += self.peek()
            self.advance()
        
        if self.peek() == quote:
            self.advance()
        else:
            raise LexerError("Unterminated string", self.line, self.column)
            
        return Token(TokenType.STRING, value, self.line, start_col)

    def handle_symbol(self):
        start_col = self.column
        char = self.peek()
        self.advance()
        
        token_type = None
        
        # Two-char operators
        if char == '|' and self.peek() == '>':
            self.advance()
            token_type = TokenType.PIPE
        elif char == '=' and self.peek() == '=':
            self.advance()
            token_type = TokenType.EQEQ
        elif char == '!' and self.peek() == '=':
            self.advance()
            token_type = TokenType.NEQ
        elif char == '<' and self.peek() == '=':
            self.advance()
            token_type = TokenType.LTE
        elif char == '>' and self.peek() == '=':
            self.advance()
            token_type = TokenType.GTE
        elif char == '-' and self.peek() == '>':
            self.advance()
            token_type = TokenType.ARROW
            
        # Single-char operators
        elif char == '+': token_type = TokenType.PLUS
        elif char == '-': token_type = TokenType.MINUS
        elif char == '*': token_type = TokenType.STAR
        elif char == '/': token_type = TokenType.SLASH
        elif char == '=': token_type = TokenType.EQ
        elif char == '<': token_type = TokenType.LT
        elif char == '>': token_type = TokenType.GT
        elif char == '(': token_type = TokenType.LPAREN
        elif char == ')': token_type = TokenType.RPAREN
        elif char == '{': token_type = TokenType.LBRACE
        elif char == '}': token_type = TokenType.RBRACE
        elif char == '[': token_type = TokenType.LBRACKET
        elif char == ']': token_type = TokenType.RBRACKET
        elif char == ',': token_type = TokenType.COMMA
        elif char == ':': token_type = TokenType.COLON
        elif char == '.': token_type = TokenType.DOT
        
        if token_type:
            self.tokens.append(Token(token_type, char, self.line, start_col))
        else:
            raise LexerError(f"Unexpected character: {char}", self.line, start_col)

    def handle_eof(self):
        # Emit NEWLINE if not just emitted?
        if self.tokens and self.tokens[-1].type != TokenType.NEWLINE and self.tokens[-1].type != TokenType.DEDENT:
             self.tokens.append(Token(TokenType.NEWLINE, line=self.line, column=1))
             
        # Dedent remaining
        while len(self.indent_stack) > 1:
            self.indent_stack.pop()
            self.tokens.append(Token(TokenType.DEDENT, line=self.line, column=1))
            
        self.tokens.append(Token(TokenType.EOF, line=self.line, column=1))
