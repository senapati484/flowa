import unittest
from flowa.interpreter import Interpreter
from flowa.lexer import Lexer
from flowa.parser import Parser

class TestStdlib(unittest.TestCase):
    def run_code(self, code):
        lexer = Lexer(code)
        tokens = lexer.tokenize()
        parser = Parser(tokens)
        module = parser.parse()
        interpreter = Interpreter()
        interpreter.interpret(module)
        return interpreter

    def test_builtins(self):
        code = """
l = list(map([1, 2, 3], str))
s = sum([1, 2, 3])
n = len(l)
i = int("42")
f = float("3.14")
"""
        interpreter = self.run_code(code)
        self.assertEqual(interpreter.globals.get("l"), ["1", "2", "3"])
        self.assertEqual(interpreter.globals.get("s"), 6)
        self.assertEqual(interpreter.globals.get("n"), 3)
        self.assertEqual(interpreter.globals.get("i"), 42)
        self.assertEqual(interpreter.globals.get("f"), 3.14)

if __name__ == '__main__':
    unittest.main()
