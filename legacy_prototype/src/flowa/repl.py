import sys
from .lexer import Lexer
from .parser import Parser
from .interpreter import Interpreter

def run_repl():
    interpreter = Interpreter()
    print("Flowa REPL v0.1")
    print("Type 'exit' to quit.")
    
    while True:
        try:
            line = input(">>> ")
            if line == "exit":
                break
            
            # Handle multi-line input (basic)
            if line.strip().endswith(":"):
                lines = [line]
                while True:
                    subline = input("... ")
                    if subline == "":
                        break
                    lines.append(subline)
                line = "\n".join(lines)
            
            lexer = Lexer(line)
            tokens = lexer.tokenize()
            parser = Parser(tokens)
            module = parser.parse()
            interpreter.interpret(module)
            
        except EOFError:
            break
        except Exception as e:
            print(f"Error: {e}")

if __name__ == "__main__":
    run_repl()
