from typing import List, Optional
from .tokens import Token, TokenType
from . import ast_nodes as ast

class ParserError(Exception):
    def __init__(self, message, token):
        super().__init__(f"{message} at {token}")
        self.token = token

class Parser:
    def __init__(self, tokens: List[Token]):
        self.tokens = tokens
        self.pos = 0

    def parse(self) -> ast.Module:
        body = []
        while not self.is_at_end():
            stmt = self.declaration()
            if stmt:
                body.append(stmt)
        return ast.Module(body=body)

    def declaration(self):
        if self.match(TokenType.DEF):
            return self.function_def(is_async=False)
        if self.match(TokenType.ASYNC):
            if self.match(TokenType.DEF):
                return self.function_def(is_async=True)
            else:
                # Could be async block or something else, but for now assume func def
                raise ParserError("Expected 'def' after 'async'", self.peek())
        if self.match(TokenType.SPAWN):
            # Spawn is an expression, but can be used as a statement?
            # Actually spawn is usually `spawn async def ...` or `spawn func()`
            # Let's treat it as an expression statement for now if it appears here.
            self.backtrack()
            return self.statement()
            
        return self.statement()

    def function_def(self, is_async: bool):
        name = self.consume(TokenType.IDENTIFIER, "Expected function name").value
        self.consume(TokenType.LPAREN, "Expected '(' after function name")
        args = []
        if not self.check(TokenType.RPAREN):
            while True:
                args.append(self.consume(TokenType.IDENTIFIER, "Expected parameter name").value)
                if not self.match(TokenType.COMMA):
                    break
        self.consume(TokenType.RPAREN, "Expected ')' after parameters")
        self.consume(TokenType.COLON, "Expected ':' before function body")
        self.consume(TokenType.NEWLINE, "Expected newline after function header")
        
        body = self.block()
        return ast.FunctionDef(name, args, body, is_async)

    def block(self) -> List[ast.ASTNode]:
        self.consume(TokenType.INDENT, "Expected indentation")
        statements = []
        while not self.check(TokenType.DEDENT) and not self.is_at_end():
            stmt = self.declaration()
            if stmt:
                statements.append(stmt)
        self.consume(TokenType.DEDENT, "Expected dedent")
        return statements

    def statement(self):
        if self.match(TokenType.RETURN):
            return self.return_stmt()
        if self.match(TokenType.IF):
            return self.if_stmt()
        if self.match(TokenType.WHILE):
            return self.while_stmt()
        if self.match(TokenType.NEWLINE):
            return None # Empty statement
            
        # Assignment or Expression Statement
        expr = self.expression()
        
        if self.match(TokenType.EQ):
            if isinstance(expr, ast.Name):
                value = self.expression()
                self.consume(TokenType.NEWLINE, "Expected newline after assignment")
                return ast.Assign(expr.id, value)
            else:
                raise ParserError("Invalid assignment target", self.peek())
        
        self.consume(TokenType.NEWLINE, "Expected newline after expression")
        return expr

    def return_stmt(self):
        value = None
        if not self.check(TokenType.NEWLINE):
            value = self.expression()
        self.consume(TokenType.NEWLINE, "Expected newline after return")
        return ast.Return(value)

    def if_stmt(self):
        test = self.expression()
        self.consume(TokenType.COLON, "Expected ':' after if condition")
        self.consume(TokenType.NEWLINE, "Expected newline after if header")
        body = self.block()
        orelse = []
        if self.match(TokenType.ELSE):
            self.consume(TokenType.COLON, "Expected ':' after else")
            self.consume(TokenType.NEWLINE, "Expected newline after else header")
            orelse = self.block()
        return ast.If(test, body, orelse)

    def while_stmt(self):
        test = self.expression()
        self.consume(TokenType.COLON, "Expected ':' after while condition")
        self.consume(TokenType.NEWLINE, "Expected newline after while header")
        body = self.block()
        return ast.While(test, body)

    def expression(self):
        return self.pipeline()

    def pipeline(self):
        expr = self.equality()
        while self.match(TokenType.PIPE):
            right = self.equality() # Should be a call usually
            expr = ast.Pipeline(expr, right)
        return expr

    def equality(self):
        expr = self.comparison()
        while self.match(TokenType.EQEQ, TokenType.NEQ):
            op = self.previous().type
            right = self.comparison()
            expr = ast.BinOp(expr, str(op), right)
        return expr

    def comparison(self):
        expr = self.term()
        while self.match(TokenType.GT, TokenType.GTE, TokenType.LT, TokenType.LTE):
            op = self.previous().type
            right = self.term()
            expr = ast.BinOp(expr, str(op), right)
        return expr

    def term(self):
        expr = self.factor()
        while self.match(TokenType.MINUS, TokenType.PLUS):
            op = self.previous().type
            right = self.factor()
            expr = ast.BinOp(expr, str(op), right)
        return expr

    def factor(self):
        expr = self.unary()
        while self.match(TokenType.SLASH, TokenType.STAR):
            op = self.previous().type
            right = self.unary()
            expr = ast.BinOp(expr, str(op), right)
        return expr

    def unary(self):
        if self.match(TokenType.SPAWN):
            return ast.Spawn(self.call()) # Expect call after spawn
        if self.match(TokenType.AWAIT):
            return ast.Await(self.unary())
        if self.match(TokenType.MINUS):
            return ast.BinOp(ast.Literal(0), '-', self.unary())
        return self.call()

    def call(self):
        expr = self.primary()
        while True:
            if self.match(TokenType.LPAREN):
                args = []
                if not self.check(TokenType.RPAREN):
                    while True:
                        args.append(self.expression())
                        if not self.match(TokenType.COMMA):
                            break
                self.consume(TokenType.RPAREN, "Expected ')' after arguments")
                expr = ast.Call(expr, args)
            else:
                break
        return expr

    def primary(self):
        if self.match(TokenType.FALSE): return ast.Literal(False)
        if self.match(TokenType.TRUE): return ast.Literal(True)
        if self.match(TokenType.NONE): return ast.Literal(None)
        
        if self.match(TokenType.NUMBER):
            return ast.Literal(self.previous().value)
        if self.match(TokenType.STRING):
            return ast.Literal(self.previous().value)
        if self.match(TokenType.IDENTIFIER):
            return ast.Name(self.previous().value)
        
        if self.match(TokenType.LPAREN):
            expr = self.expression()
            self.consume(TokenType.RPAREN, "Expected ')' after expression")
            return expr
            
        if self.match(TokenType.LBRACKET):
            elements = []
            if not self.check(TokenType.RBRACKET):
                while True:
                    elements.append(self.expression())
                    if not self.match(TokenType.COMMA):
                        break
            self.consume(TokenType.RBRACKET, "Expected ']' after list elements")
            # We need a List AST node or just use Literal for now?
            # My AST has Literal(value). value can be a list of values?
            # But elements are AST nodes. We need a List AST node.
            # Let's check ast_nodes.py.
            # I don't have a List node. I have Literal.
            # If I use Literal, I need to evaluate elements first?
            # No, AST should represent structure.
            # I should add a List node to AST.
            # For now, let's hack it: return a Call to 'list' with elements?
            # Or just add List node. Adding List node is better.
            return ast.List(elements)

        if self.match(TokenType.LAMBDA):
            # Simple lambda: lambda x, y: x + y
            args = []
            if not self.check(TokenType.COLON):
                while True:
                    args.append(self.consume(TokenType.IDENTIFIER, "Expected parameter name").value)
                    if not self.match(TokenType.COMMA):
                        break
            self.consume(TokenType.COLON, "Expected ':' after lambda args")
            body = self.expression()
            return ast.Lambda(args, body)

        raise ParserError("Expect expression", self.peek())

    # Helper methods
    def match(self, *types):
        for type in types:
            if self.check(type):
                self.advance()
                return True
        return False

    def check(self, type):
        if self.is_at_end(): return False
        return self.peek().type == type

    def advance(self):
        if not self.is_at_end():
            self.pos += 1
        return self.previous()

    def is_at_end(self):
        return self.peek().type == TokenType.EOF

    def peek(self):
        return self.tokens[self.pos]

    def previous(self):
        return self.tokens[self.pos - 1]

    def consume(self, type, message):
        if self.check(type):
            return self.advance()
        raise ParserError(message, self.peek())
    
    def backtrack(self):
        self.pos -= 1
