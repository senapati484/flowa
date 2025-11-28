from enum import Enum, auto
from dataclasses import dataclass
from typing import Any

class TokenType(Enum):
    # Special
    EOF = auto()
    INDENT = auto()
    DEDENT = auto()
    NEWLINE = auto()
    
    # Identifiers & Literals
    IDENTIFIER = auto()
    NUMBER = auto()
    STRING = auto()
    
    # Keywords
    DEF = auto()
    ASYNC = auto()
    RETURN = auto()
    IF = auto()
    ELSE = auto()
    WHILE = auto()
    FOR = auto()
    IN = auto()
    SPAWN = auto()
    AWAIT = auto()
    TRUE = auto()
    FALSE = auto()
    NONE = auto()
    LAMBDA = auto()
    
    # Operators
    PIPE = auto()       # |>
    PLUS = auto()       # +
    MINUS = auto()      # -
    STAR = auto()       # *
    SLASH = auto()      # /
    EQ = auto()         # =
    EQEQ = auto()       # ==
    NEQ = auto()        # !=
    LT = auto()         # <
    GT = auto()         # >
    LTE = auto()        # <=
    GTE = auto()        # >=
    LPAREN = auto()     # (
    RPAREN = auto()     # )
    LBRACE = auto()     # {
    RBRACE = auto()     # }
    LBRACKET = auto()   # [
    RBRACKET = auto()   # ]
    COMMA = auto()      # ,
    COLON = auto()      # :
    DOT = auto()        # .
    ARROW = auto()      # ->

@dataclass
class Token:
    type: TokenType
    value: Any = None
    line: int = 0
    column: int = 0
    
    def __repr__(self):
        return f"Token({self.type.name}, {self.value!r}, line={self.line})"
