import unittest
from flowa.lexer import Lexer
from flowa.parser import Parser
from flowa import ast_nodes as ast

class TestParser(unittest.TestCase):
    def parse(self, code):
        lexer = Lexer(code)
        tokens = lexer.tokenize()
        parser = Parser(tokens)
        return parser.parse()

    def test_basic_assign(self):
        code = "x = 10"
        module = self.parse(code)
        self.assertIsInstance(module, ast.Module)
        self.assertEqual(len(module.body), 1)
        assign = module.body[0]
        self.assertIsInstance(assign, ast.Assign)
        self.assertEqual(assign.target, "x")
        self.assertIsInstance(assign.value, ast.Literal)
        self.assertEqual(assign.value.value, 10)

    def test_function_def(self):
        code = """
def add(a, b):
    return a + b
"""
        module = self.parse(code)
        func = module.body[0]
        self.assertIsInstance(func, ast.FunctionDef)
        self.assertEqual(func.name, "add")
        self.assertEqual(func.args, ["a", "b"])
        self.assertIsInstance(func.body[0], ast.Return)

    def test_pipeline(self):
        code = "data |> map(f)"
        module = self.parse(code)
        stmt = module.body[0]
        self.assertIsInstance(stmt, ast.Pipeline)
        self.assertIsInstance(stmt.left, ast.Name)
        self.assertEqual(stmt.left.id, "data")
        self.assertIsInstance(stmt.right, ast.Call)

    def test_precedence(self):
        code = "x = 1 + 2 * 3"
        module = self.parse(code)
        assign = module.body[0]
        # 1 + (2 * 3)
        self.assertIsInstance(assign.value, ast.BinOp)
        self.assertEqual(assign.value.op, "TokenType.PLUS")
        self.assertIsInstance(assign.value.right, ast.BinOp)
        self.assertEqual(assign.value.right.op, "TokenType.STAR")

if __name__ == '__main__':
    unittest.main()
