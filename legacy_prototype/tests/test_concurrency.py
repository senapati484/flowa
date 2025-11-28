import unittest
import asyncio
from flowa.lexer import Lexer
from flowa.parser import Parser
from flowa.interpreter import Interpreter

class TestConcurrency(unittest.TestCase):
    def run_async_code(self, code):
        lexer = Lexer(code)
        tokens = lexer.tokenize()
        parser = Parser(tokens)
        module = parser.parse()
        interpreter = Interpreter()
        
        # We need to run the interpreter's async execution
        # But interpret() is sync.
        # We need a way to run async code from top level or wrap in spawn.
        
        # Let's wrap everything in a main async function and spawn it?
        # Or just use interpreter.execute_block_async manually?
        
        async def run():
            # We need to execute the module body. 
            # But module body is a list of statements.
            # Some might be async (spawn), some sync.
            # If we treat the top level as an async block:
            await interpreter.execute_block_async(module.body, interpreter.globals)
            return interpreter
            
        return asyncio.run(run())

    def test_spawn_await(self):
        code = """
async def get_value():
    return 42

async def main():
    task = spawn get_value()
    result = await task
    return result

# We need to call main. But we can't call it from top level sync code easily in this test setup
# unless we hack the test runner.
# Let's just define main and then call it in our python test code.
"""
        # Run the code to define functions
        interpreter = self.run_async_code(code)
        
        # Now call main() manually
        main_func = interpreter.globals.get("main")
        
        async def call_main():
            return await main_func.call(interpreter, [])
            
        result = asyncio.run(call_main())
        self.assertEqual(result, 42)

    def test_concurrency_execution(self):
        # Test that things actually run concurrently?
        # Hard to test deterministically without sleeps.
        # But we can test that we can spawn multiple things.
        code = """
async def add(a, b):
    return a + b

async def main():
    t1 = spawn add(10, 20)
    t2 = spawn add(30, 40)
    r1 = await t1
    r2 = await t2
    return r1 + r2
"""
        interpreter = self.run_async_code(code)
        main_func = interpreter.globals.get("main")
        
        async def call_main():
            return await main_func.call(interpreter, [])
            
        result = asyncio.run(call_main())
        self.assertEqual(result, 100)

if __name__ == '__main__':
    unittest.main()
