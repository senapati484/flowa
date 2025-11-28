import asyncio
from typing import Any, Dict, List, Optional, Callable
from . import ast_nodes as ast
from .tokens import TokenType

class Environment:
    def __init__(self, parent=None):
        self.values: Dict[str, Any] = {}
        self.parent: Optional[Environment] = parent

    def define(self, name: str, value: Any):
        self.values[name] = value

    def get(self, name: str) -> Any:
        if name in self.values:
            return self.values[name]
        if self.parent:
            return self.parent.get(name)
        raise RuntimeError(f"Undefined variable '{name}'")

    def assign(self, name: str, value: Any):
        if name in self.values:
            self.values[name] = value
            return
        if self.parent:
            self.parent.assign(name, value)
            return
        raise RuntimeError(f"Undefined variable '{name}'")

class Function:
    def __init__(self, declaration: ast.FunctionDef, closure: Environment):
        self.declaration = declaration
        self.closure = closure

    def call(self, interpreter, args: List[Any]):
        environment = Environment(self.closure)
        for i, arg in enumerate(args):
            environment.define(self.declaration.args[i], arg)
        
        try:
            interpreter.execute_block(self.declaration.body, environment)
        except ReturnException as e:
            return e.value
        return None

class AsyncFunction(Function):
    async def call(self, interpreter, args: List[Any]):
        environment = Environment(self.closure)
        for i, arg in enumerate(args):
            environment.define(self.declaration.args[i], arg)
        
        try:
            await interpreter.execute_block_async(self.declaration.body, environment)
        except ReturnException as e:
            return e.value
        return None

class ReturnException(Exception):
    def __init__(self, value):
        self.value = value

class Interpreter:
    def __init__(self):
        self.globals = Environment()
        self.environment = self.globals
        
        # Built-ins
        def flowa_map(data, func):
            def wrapper(arg):
                if isinstance(func, Function):
                    return func.call(self, [arg])
                return func(arg)
            return map(wrapper, data)
        
        def flowa_filter(data, func):
            def wrapper(arg):
                if isinstance(func, Function):
                    # Filter expects boolean, ensure we get it?
                    # Flowa doesn't strictly enforce types, but Python filter does for truthiness.
                    return func.call(self, [arg])
                return func(arg)
            return filter(wrapper, data)

        self.globals.define("print", print)
        self.globals.define("input", input)
        self.globals.define("map", flowa_map)
        self.globals.define("filter", flowa_filter)
        self.globals.define("sum", sum)
        self.globals.define("list", list)
        self.globals.define("int", int)
        self.globals.define("float", float)
        self.globals.define("str", str)
        self.globals.define("len", len)


    def interpret(self, module: ast.Module):
        try:
            for stmt in module.body:
                self.execute(stmt)
        except Exception as e:
            print(f"Runtime Error: {e}")

    def execute(self, stmt: ast.ASTNode):
        if isinstance(stmt, ast.FunctionDef):
            func = AsyncFunction(stmt, self.environment) if stmt.is_async else Function(stmt, self.environment)
            self.environment.define(stmt.name, func)
        elif isinstance(stmt, ast.Return):
            value = None
            if stmt.value:
                value = self.evaluate(stmt.value)
            raise ReturnException(value)
        elif isinstance(stmt, ast.Assign):
            value = self.evaluate(stmt.value)
            self.environment.define(stmt.target, value) # Simple define for now, should check assign
        elif isinstance(stmt, ast.If):
            if self.evaluate(stmt.test):
                self.execute_block(stmt.body, Environment(self.environment))
            elif stmt.orelse:
                self.execute_block(stmt.orelse, Environment(self.environment))
        elif isinstance(stmt, ast.While):
            while self.evaluate(stmt.test):
                self.execute_block(stmt.body, Environment(self.environment))
        elif isinstance(stmt, ast.Expr):
            self.evaluate(stmt)

    def execute_block(self, statements: List[ast.ASTNode], environment: Environment):
        previous = self.environment
        try:
            self.environment = environment
            for stmt in statements:
                self.execute(stmt)
        finally:
            self.environment = previous

    async def execute_block_async(self, statements: List[ast.ASTNode], environment: Environment):
        previous = self.environment
        try:
            self.environment = environment
            for stmt in statements:
                await self.execute_async(stmt)
        finally:
            self.environment = previous

    async def execute_async(self, stmt: ast.ASTNode):
        if isinstance(stmt, ast.Return):
            value = None
            if stmt.value:
                value = await self.evaluate_async(stmt.value)
            raise ReturnException(value)
        elif isinstance(stmt, ast.Assign):
            value = await self.evaluate_async(stmt.value)
            self.environment.define(stmt.target, value)
        elif isinstance(stmt, ast.If):
            if await self.evaluate_async(stmt.test):
                await self.execute_block_async(stmt.body, Environment(self.environment))
            elif stmt.orelse:
                await self.execute_block_async(stmt.orelse, Environment(self.environment))
        elif isinstance(stmt, ast.While):
            while await self.evaluate_async(stmt.test):
                await self.execute_block_async(stmt.body, Environment(self.environment))
        elif isinstance(stmt, ast.FunctionDef):
            self.execute(stmt) # Function def is sync
        elif isinstance(stmt, ast.Expr):
            await self.evaluate_async(stmt)

    def evaluate(self, expr: ast.ASTNode) -> Any:
        if isinstance(expr, ast.Literal):
            return expr.value
        elif isinstance(expr, ast.Name):
            return self.environment.get(expr.id)
        elif isinstance(expr, ast.BinOp):
            left = self.evaluate(expr.left)
            right = self.evaluate(expr.right)
            return self.apply_op(expr.op, left, right)
        elif isinstance(expr, ast.Call):
            callee = self.evaluate(expr.func)
            args = [self.evaluate(arg) for arg in expr.args]
            if isinstance(callee, Function):
                return callee.call(self, args)
            elif callable(callee):
                return callee(*args)
            else:
                raise RuntimeError("Not a function")
        elif isinstance(expr, ast.Pipeline):
            # left |> right(args) -> right(left, args)
            # left |> right -> right(left)
            left_val = self.evaluate(expr.left)
            if isinstance(expr.right, ast.Call):
                # Inject left_val as first arg
                callee = self.evaluate(expr.right.func)
                args = [left_val] + [self.evaluate(arg) for arg in expr.right.args]
                if isinstance(callee, Function):
                    return callee.call(self, args)
                elif callable(callee):
                    return callee(*args)
            elif isinstance(expr.right, ast.Name):
                callee = self.evaluate(expr.right)
                if callable(callee) or isinstance(callee, Function):
                    if isinstance(callee, Function):
                        return callee.call(self, [left_val])
                    return callee(left_val)
            raise RuntimeError("Invalid pipeline right hand side")
        elif isinstance(expr, ast.Lambda):
            # Create a Function object for the lambda
            # Lambda body is an expression, but Function expects a block (list of statements).
            # We need to wrap the body in a Return statement.
            body = [ast.Return(expr.body)]
            # Create a dummy FunctionDef
            func_def = ast.FunctionDef(name="<lambda>", args=expr.args, body=body, is_async=False)
            return Function(func_def, self.environment)
        elif isinstance(expr, ast.List):
            return [self.evaluate(e) for e in expr.elements]
        return None

    async def evaluate_async(self, expr: ast.ASTNode) -> Any:
        if isinstance(expr, ast.Await):
            # await value
            val = await self.evaluate_async(expr.value)
            if isinstance(val, asyncio.Future) or asyncio.iscoroutine(val):
                return await val
            return val
        elif isinstance(expr, ast.Spawn):
            # spawn call -> return Future
            # We need to schedule the coroutine.
            # expr.call must be a call to an async function.
            # But evaluate(expr.call) would try to execute it?
            # No, calling AsyncFunction returns a coroutine (if we implemented it that way)
            # My AsyncFunction.call is async def, so it returns a coroutine.
            
            # We need to evaluate the call manually to get the coroutine without awaiting it?
            # Actually, `evaluate` for Call on AsyncFunction should probably return the coroutine if not awaited?
            # But `evaluate` is sync.
            
            # Let's handle Spawn specifically.
            if isinstance(expr.call, ast.Call):
                callee = await self.evaluate_async(expr.call.func)
                args = [await self.evaluate_async(arg) for arg in expr.call.args]
                if isinstance(callee, AsyncFunction):
                    coro = callee.call(self, args)
                    return asyncio.create_task(coro)
            raise RuntimeError("Spawn requires a call to an async function")
            
        # ... handle other nodes recursively with await ...
        # For MVP, let's duplicate logic or delegate to evaluate if no await involved?
        # But sub-expressions might have await.
        
        if isinstance(expr, ast.BinOp):
            left = await self.evaluate_async(expr.left)
            right = await self.evaluate_async(expr.right)
            return self.apply_op(expr.op, left, right)
        elif isinstance(expr, ast.Call):
            callee = await self.evaluate_async(expr.func)
            args = [await self.evaluate_async(arg) for arg in expr.args]
            if isinstance(callee, AsyncFunction):
                return await callee.call(self, args)
            elif isinstance(callee, Function):
                return callee.call(self, args)
            elif callable(callee):
                if asyncio.iscoroutinefunction(callee):
                    return await callee(*args)
                return callee(*args)
        
        elif isinstance(expr, ast.List):
            return [await self.evaluate_async(e) for e in expr.elements]

        return self.evaluate(expr) # Fallback for simple nodes

    def apply_op(self, op, left, right):
        if op == 'TokenType.PLUS' or op == '+': return left + right
        if op == 'TokenType.MINUS' or op == '-': return left - right
        if op == 'TokenType.STAR' or op == '*': return left * right
        if op == 'TokenType.SLASH' or op == '/': return left / right
        if op == 'TokenType.GT' or op == '>': return left > right
        if op == 'TokenType.LT' or op == '<': return left < right
        if op == 'TokenType.EQEQ' or op == '==': return left == right
        raise RuntimeError(f"Unknown operator {op}")
