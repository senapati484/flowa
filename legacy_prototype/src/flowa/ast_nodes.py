from dataclasses import dataclass
from typing import List, Optional, Any, Union

@dataclass
class ASTNode:
    pass

@dataclass
class Module(ASTNode):
    body: List[ASTNode]

@dataclass
class FunctionDef(ASTNode):
    name: str
    args: List[str]
    body: List[ASTNode]
    is_async: bool = False

@dataclass
class Return(ASTNode):
    value: Optional[ASTNode]

@dataclass
class If(ASTNode):
    test: ASTNode
    body: List[ASTNode]
    orelse: List[ASTNode]

@dataclass
class While(ASTNode):
    test: ASTNode
    body: List[ASTNode]

@dataclass
class Assign(ASTNode):
    target: str
    value: ASTNode

@dataclass
class Expr(ASTNode):
    pass

@dataclass
class BinOp(Expr):
    left: ASTNode
    op: str
    right: ASTNode

@dataclass
class Call(Expr):
    func: ASTNode
    args: List[ASTNode]

@dataclass
class Pipeline(Expr):
    left: ASTNode
    right: ASTNode # Usually a Call

@dataclass
class Spawn(Expr):
    call: Call

@dataclass
class Await(Expr):
    value: ASTNode

@dataclass
class Literal(Expr):
    value: Any

@dataclass
class Name(Expr):
    id: str

@dataclass
class Lambda(Expr):
    args: List[str]
    body: ASTNode

@dataclass
class List(Expr):
    elements: List[ASTNode]

