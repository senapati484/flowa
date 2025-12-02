package opcode

import (
	"fmt"
)

type Opcode byte

type Instructions []byte

const (
	// OpConstant retrieves a constant from the constant pool
	OpConstant Opcode = iota
	// OpAdd adds the top two elements of the stack
	OpAdd
	// OpSub subtracts the top two elements of the stack
	OpSub
	// OpMul multiplies the top two elements of the stack
	OpMul
	// OpDiv divides the top two elements of the stack
	OpDiv
	// OpPop pops the top element of the stack
	OpPop
	// OpTrue pushes true onto the stack
	OpTrue
	// OpFalse pushes false onto the stack
	OpFalse
	// OpEqual compares the top two elements for equality
	OpEqual
	// OpNotEqual compares the top two elements for inequality
	OpNotEqual
	// OpGreaterThan compares the top two elements for greater than
	OpGreaterThan
	// OpMinus negates the top element of the stack
	OpMinus
	// OpBang negates the boolean value of the top element of the stack
	OpBang
	// OpJumpNotTruth jumps to the operand address if the top element is not truthy
	OpJumpNotTruth
	// OpJump jumps to the operand address
	OpJump
	// OpNull pushes null onto the stack
	OpNull
	// OpGetGlobal retrieves a global variable
	OpGetGlobal
	// OpSetGlobal sets a global variable
	OpSetGlobal
	// OpArray creates an array from the top N elements of the stack
	OpArray
	// OpHash creates a hash map from the top 2*N elements of the stack
	OpHash
	// OpIndex retrieves an element from an indexable object
	OpIndex
	// OpCall calls a function
	OpCall
	// OpReturnValue returns a value from a function
	OpReturnValue
	// OpReturn returns null from a function
	OpReturn
	// OpGetLocal retrieves a local variable
	OpGetLocal
	// OpSetLocal sets a local variable
	OpSetLocal
	// OpGetBuiltin retrieves a builtin function
	OpGetBuiltin
	// OpIncLocal increments a local variable by 1
	OpIncLocal
	// OpAddLocal adds a local variable to another local variable
	OpAddLocal
	// OpJumpIfLocalGreaterEqualConst jumps if local >= const
	OpJumpIfLocalGreaterEqualConst
)

type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant:                     {"OpConstant", []int{2}},
	OpAdd:                          {"OpAdd", []int{}},
	OpSub:                          {"OpSub", []int{}},
	OpMul:                          {"OpMul", []int{}},
	OpDiv:                          {"OpDiv", []int{}},
	OpPop:                          {"OpPop", []int{}},
	OpTrue:                         {"OpTrue", []int{}},
	OpFalse:                        {"OpFalse", []int{}},
	OpEqual:                        {"OpEqual", []int{}},
	OpNotEqual:                     {"OpNotEqual", []int{}},
	OpGreaterThan:                  {"OpGreaterThan", []int{}},
	OpMinus:                        {"OpMinus", []int{}},
	OpBang:                         {"OpBang", []int{}},
	OpJumpNotTruth:                 {"OpJumpNotTruth", []int{2}},
	OpJump:                         {"OpJump", []int{2}},
	OpNull:                         {"OpNull", []int{}},
	OpGetGlobal:                    {"OpGetGlobal", []int{2}},
	OpSetGlobal:                    {"OpSetGlobal", []int{2}},
	OpArray:                        {"OpArray", []int{2}},
	OpHash:                         {"OpHash", []int{2}},
	OpIndex:                        {"OpIndex", []int{}},
	OpCall:                         {"OpCall", []int{1}},
	OpReturnValue:                  {"OpReturnValue", []int{}},
	OpReturn:                       {"OpReturn", []int{}},
	OpGetLocal:                     {"OpGetLocal", []int{1}},
	OpSetLocal:                     {"OpSetLocal", []int{1}},
	OpGetBuiltin:                   {"OpGetBuiltin", []int{1}},
	OpIncLocal:                     {"OpIncLocal", []int{1}},
	OpAddLocal:                     {"OpAddLocal", []int{1, 1}},
	OpJumpIfLocalGreaterEqualConst: {"OpJumpIfLocalGreaterEqualConst", []int{1, 2, 2}},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}
	return def, nil
}

func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	instructionLen := 1
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 2:
			instruction[offset] = byte(o >> 8)
			instruction[offset+1] = byte(o)
		case 1:
			instruction[offset] = byte(o)
		}
		offset += width
	}

	return instruction
}

func ReadOperands(def *Definition, ins []byte) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0

	for i, width := range def.OperandWidths {
		switch width {
		case 2:
			operands[i] = int(ReadUint16(ins[offset:]))
		case 1:
			operands[i] = int(ReadUint8(ins[offset:]))
		}
		offset += width
	}

	return operands, offset
}

func ReadUint16(ins []byte) uint16 {
	return uint16(ins[0])<<8 | uint16(ins[1])
}

func ReadUint8(ins []byte) uint8 {
	return uint8(ins[0])
}

func (ins Opcode) String() string {
	def, ok := definitions[ins]
	if !ok {
		return fmt.Sprintf("Opcode(%d)", ins)
	}
	return def.Name
}
