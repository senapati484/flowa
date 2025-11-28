import sys
import argparse
from .lexer import Lexer
from .parser import Parser
from .interpreter import Interpreter
from .repl import run_repl

def run_file(path):
    with open(path, 'r') as f:
        code = f.read()
    
    lexer = Lexer(code)
    tokens = lexer.tokenize()
    parser = Parser(tokens)
    module = parser.parse()
    interpreter = Interpreter()
    interpreter.interpret(module)

def main():
    parser = argparse.ArgumentParser(description="Flowa Language")
    parser.add_argument('script', nargs='?', help="Script file to run")
    args = parser.parse_args()
    
    if args.script:
        run_file(args.script)
    else:
        run_repl()

if __name__ == "__main__":
    main()
