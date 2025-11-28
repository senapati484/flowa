import unittest
import asyncio
from flowa.lexer import Lexer
from flowa.parser import Parser
from flowa.interpreter import Interpreter

class TestInterpreter(unittest.TestCase):
    def run_code(self, code):
        lexer = Lexer(code)
        tokens = lexer.tokenize()
        parser = Parser(tokens)
        module = parser.parse()
        interpreter = Interpreter()
        interpreter.interpret(module)
        return interpreter

    def test_basic_arithmetic(self):
        code = """
x = 10 + 20 * 2
"""
        interpreter = self.run_code(code)
        self.assertEqual(interpreter.globals.get("x"), 50)

    def test_function(self):
        code = """
def add(a, b):
    return a + b

result = add(10, 5)
"""
        interpreter = self.run_code(code)
        self.assertEqual(interpreter.globals.get("result"), 15)

    def test_pipeline(self):
        code = """
def double(x):
    return x * 2

result = 10 |> double()
"""
        interpreter = self.run_code(code)
        self.assertEqual(interpreter.globals.get("result"), 20)

    def test_recursion(self):
        code = """
def fib(n):
    if n < 2:
        return n
    return fib(n-1) + fib(n-2)

result = fib(10)
"""
        interpreter = self.run_code(code)
        self.assertEqual(interpreter.globals.get("result"), 55)

    def test_async(self):
        # We need to run the interpreter in an async loop for this test?
        # My interpreter.interpret is sync, but it can handle async functions if called?
        # Wait, interpret calls execute, which handles AsyncFunction definition.
        # But to call an async function, we need an entry point that awaits.
        # Let's try to define an async function and call it from another async function?
        # Or just test that we can define it.
        code = """
async def foo():
    return 42

# We can't call async function from top level sync code in my MVP interpreter yet
# unless we use spawn?
"""
        interpreter = self.run_code(code)
        # Check if foo is defined
        foo = interpreter.globals.get("foo")
        self.assertTrue(hasattr(foo, 'call')) # It's an AsyncFunction instance

if __name__ == '__main__':
    unittest.main()
