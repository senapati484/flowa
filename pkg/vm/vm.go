package vm

import (
	"flowa/pkg/ast"
	"flowa/pkg/compiler"
	"flowa/pkg/eval"
	"flowa/pkg/opcode"
	"fmt"
	"time"
)

const StackSize = 256   // Reduced from 2048 for better cache locality
const GlobalsSize = 256 // Reduced from 65536 - 256 globals should be sufficient
const MaxFrames = 64    // Reduced from 1024 - supports 64 levels of function calls

// Frame represents a call frame
type Frame struct {
	fn          *eval.Function
	ip          int // instruction pointer for this frame
	basePointer int // base pointer for local variables in this frame
}

func NewFrame(fn *eval.Function, basePointer int) *Frame {
	return &Frame{
		fn:          fn,
		ip:          -1,
		basePointer: basePointer,
	}
}

type VM struct {
	constants []eval.Object
	globals   []eval.Object

	stack []eval.Object
	sp    int // Always points to the next value. Top of stack is stack[sp-1]

	builtins []eval.Object

	frames      []*Frame
	framesIndex int
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &eval.Function{Body: &ast.BlockStatement{}, Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	// Initialize builtins
	builtins := make([]eval.Object, 3)

	// 0: print
	builtins[0] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(arg.Inspect())
			}
			fmt.Println()
			return eval.NULL
		},
	}

	// 1: len
	builtins[1] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			switch arg := args[0].(type) {
			case *eval.String:
				return &eval.Integer{Value: int64(len(arg.Value))}
			case *eval.Array:
				return &eval.Integer{Value: int64(len(arg.Elements))}
			default:
				return &eval.ErrorObj{Message: fmt.Sprintf("argument to `len` not supported, got %s", args[0].Kind())}
			}
		},
	}

	// 2: time module
	timeEnv := eval.NewEnvironment()
	timeEnv.Set("now_ms", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			return &eval.Integer{Value: time.Now().UnixMilli()}
		},
	})
	timeEnv.Set("since_s", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 || len(args) > 2 {
				return &eval.ErrorObj{Message: "wrong number of arguments: expected 1 or 2"}
			}
			start, ok := args[0].(*eval.Integer)
			if !ok {
				return &eval.ErrorObj{Message: "first argument must be integer"}
			}

			// Default precision is 3
			precision := 3
			if len(args) == 2 {
				precisionArg, ok := args[1].(*eval.Integer)
				if !ok {
					return &eval.ErrorObj{Message: "second argument (precision) must be integer"}
				}
				precision = int(precisionArg.Value)
			}

			now := time.Now().UnixMilli()
			diff := now - start.Value
			formatStr := fmt.Sprintf("%%.%df", precision)
			return &eval.String{Value: fmt.Sprintf(formatStr, float64(diff)/1000.0)}
		},
	})
	builtins[2] = &eval.Module{Name: "time", Env: timeEnv}

	return &VM{
		constants: bytecode.Constants,
		globals:   make([]eval.Object, GlobalsSize),

		stack: make([]eval.Object, StackSize),
		sp:    bytecode.MainNumLocals,

		builtins: builtins,

		frames:      frames,
		framesIndex: 1,
	}
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

func (vm *VM) StackTop() eval.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) Run() error {
	// Cache frame pointer to avoid repeated function calls
	frame := vm.currentFrame()
	ip := frame.ip
	ins := frame.fn.Instructions

	// Cache frequently accessed fields
	stack := vm.stack
	sp := vm.sp
	constants := vm.constants

	// Main execution loop
	for ip < len(ins)-1 {
		ip++
		op := opcode.Opcode(ins[ip])

		switch op {
		case opcode.OpConstant:
			constIndex := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2
			stack[sp] = constants[constIndex]
			sp++
			// fmt.Printf("OpConstant: sp=%d\n", sp)

		case opcode.OpGetBuiltin:
			builtinIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1
			stack[sp] = vm.builtins[builtinIndex]
			sp++

		case opcode.OpIncLocal:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1

			obj := stack[frame.basePointer+localIndex]
			if intObj, ok := obj.(*eval.Integer); ok {
				// Create new integer to preserve immutability
				// TODO: Use object pooling here for further optimization
				stack[frame.basePointer+localIndex] = eval.NewInteger(intObj.Value + 1)
			} else {
				return fmt.Errorf("operand to OpIncLocal must be an integer, got %s", obj.Kind())
			}

		case opcode.OpAddLocal:
			targetIndex := int(opcode.ReadUint8(ins[ip+1:]))
			sourceIndex := int(opcode.ReadUint8(ins[ip+2:]))
			ip += 2

			targetObj := stack[frame.basePointer+targetIndex]
			sourceObj := stack[frame.basePointer+sourceIndex]

			targetInt, ok1 := targetObj.(*eval.Integer)
			sourceInt, ok2 := sourceObj.(*eval.Integer)

			if ok1 && ok2 {
				// Create new integer to preserve immutability
				stack[frame.basePointer+targetIndex] = eval.NewInteger(targetInt.Value + sourceInt.Value)
			} else {
				return fmt.Errorf("operands to OpAddLocal must be integers")
			}

		case opcode.OpIndex:
			index := stack[sp-1]
			left := stack[sp-2]

			// Sync vm.sp before calling method
			vm.sp = sp - 2

			err := vm.executeIndexExpression(left, index)
			if err != nil {
				return err
			}
			// Sync local sp back
			sp = vm.sp

		case opcode.OpAdd, opcode.OpSub, opcode.OpMul, opcode.OpDiv:
			// Inlined binary operation
			right := stack[sp-1]
			left := stack[sp-2]
			sp -= 2

			leftInt := left.(*eval.Integer)
			rightInt := right.(*eval.Integer)

			var result int64
			switch op {
			case opcode.OpAdd:
				result = leftInt.Value + rightInt.Value
			case opcode.OpSub:
				result = leftInt.Value - rightInt.Value
			case opcode.OpMul:
				result = leftInt.Value * rightInt.Value
			case opcode.OpDiv:
				result = leftInt.Value / rightInt.Value
			}

			// Inlined push
			stack[sp] = eval.NewInteger(result)
			sp++

		case opcode.OpPop:
			sp--

		case opcode.OpTrue:
			stack[sp] = eval.TRUE
			sp++

		case opcode.OpFalse:
			stack[sp] = eval.FALSE
			sp++

		case opcode.OpEqual, opcode.OpNotEqual, opcode.OpGreaterThan:
			right := stack[sp-1]
			left := stack[sp-2]
			sp -= 2

			leftInt, leftOk := left.(*eval.Integer)
			rightInt, rightOk := right.(*eval.Integer)

			var result bool
			if leftOk && rightOk {
				switch op {
				case opcode.OpEqual:
					result = leftInt.Value == rightInt.Value
				case opcode.OpNotEqual:
					result = leftInt.Value != rightInt.Value
				case opcode.OpGreaterThan:
					result = leftInt.Value > rightInt.Value
				}
			} else {
				// Boolean comparison
				switch op {
				case opcode.OpEqual:
					result = right == left
				case opcode.OpNotEqual:
					result = right != left
				}
			}

			if result {
				stack[sp] = eval.TRUE
			} else {
				stack[sp] = eval.FALSE
			}
			sp++

		case opcode.OpMinus:
			operand := stack[sp-1]
			intOperand := operand.(*eval.Integer)
			stack[sp-1] = eval.NewInteger(-intOperand.Value)

		case opcode.OpBang:
			operand := stack[sp-1]
			switch operand {
			case eval.TRUE:
				stack[sp-1] = eval.FALSE
			case eval.FALSE, eval.NULL:
				stack[sp-1] = eval.TRUE
			default:
				stack[sp-1] = eval.FALSE
			}

		case opcode.OpNull:
			stack[sp] = eval.NULL
			sp++

		case opcode.OpSetGlobal:
			globalIndex := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2
			sp--
			vm.globals[globalIndex] = stack[sp]

		case opcode.OpGetGlobal:
			globalIndex := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2
			stack[sp] = vm.globals[globalIndex]
			sp++

		case opcode.OpSetLocal:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1
			sp--
			stack[frame.basePointer+localIndex] = stack[sp]

		case opcode.OpGetLocal:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1
			stack[sp] = stack[frame.basePointer+localIndex]
			sp++

		case opcode.OpJump:
			pos := int(opcode.ReadUint16(ins[ip+1:]))
			ip = pos - 1

		case opcode.OpJumpIfLocalGreaterEqualConst:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			constIndex := int(opcode.ReadUint16(ins[ip+2:]))
			pos := int(opcode.ReadUint16(ins[ip+4:]))
			ip += 5

			localObj := stack[frame.basePointer+localIndex]
			constObj := constants[constIndex]

			localInt, ok1 := localObj.(*eval.Integer)
			constInt, ok2 := constObj.(*eval.Integer)

			if !ok1 || !ok2 {
				return fmt.Errorf("operands to OpJumpIfLocalGreaterEqualConst must be integers")
			}

			if localInt.Value >= constInt.Value {
				ip = pos - 1
			}

		case opcode.OpJumpNotTruth:
			pos := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2

			sp--
			condition := stack[sp]
			if !isTruthy(condition) {
				ip = pos - 1
			}

		case opcode.OpCall:
			numArgs := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1

			// Sync vm.sp before calling method
			vm.sp = sp

			// Check current frame before call
			currentFrame := frame

			err := vm.executeCall(numArgs)
			if err != nil {
				return err
			}

			// Sync local sp back
			sp = vm.sp

			// Only reload if frame changed (normal function call)
			// For builtins, we stay in the same frame and continue execution
			if vm.currentFrame() != currentFrame {
				frame = vm.currentFrame()
				ip = frame.ip
				ins = frame.fn.Instructions
				stack = vm.stack
				constants = vm.constants
			}

		case opcode.OpReturnValue:
			returnValue := stack[sp-1]
			sp--

			frame = vm.popFrame()
			vm.sp = frame.basePointer - 1

			stack[vm.sp] = returnValue
			vm.sp++

			// Reload cached values
			frame = vm.currentFrame()
			ip = frame.ip
			ins = frame.fn.Instructions
			sp = vm.sp

		case opcode.OpReturn:
			frame = vm.popFrame()
			vm.sp = frame.basePointer - 1

			stack[vm.sp] = eval.NULL
			vm.sp++

			// Reload cached values
			frame = vm.currentFrame()
			ip = frame.ip
			ins = frame.fn.Instructions
			sp = vm.sp
		}
	}

	// Save final state
	frame.ip = ip
	vm.sp = sp

	return nil
}

func (vm *VM) executeCall(numArgs int) error {
	callee := vm.stack[vm.sp-1-numArgs]
	if callee == nil {
		return fmt.Errorf("callee is nil! sp=%d, numArgs=%d", vm.sp, numArgs)
	}
	switch fn := callee.(type) {
	case *eval.Function:
		frame := NewFrame(fn, vm.sp-numArgs)
		vm.pushFrame(frame)
		vm.sp = frame.basePointer + fn.SlotCount
		return nil

	case *eval.BuiltinFunction:
		args := vm.stack[vm.sp-numArgs : vm.sp]
		result := fn.Fn(args...)
		vm.sp -= numArgs + 1
		return vm.push(result)

	default:
		return fmt.Errorf("calling non-function: %s", callee.Kind())
	}
}

func (vm *VM) push(o eval.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() eval.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) LastPoppedStackElem() eval.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) executeBinaryOperation(op opcode.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftInt, leftOk := left.(*eval.Integer)
	rightInt, rightOk := right.(*eval.Integer)

	if !leftOk || !rightOk {
		return fmt.Errorf("unsupported types for binary operation: %s %s",
			left.Kind(), right.Kind())
	}

	return vm.executeBinaryIntegerOperation(op, leftInt, rightInt)
}

func (vm *VM) executeBinaryIntegerOperation(
	op opcode.Opcode,
	left, right *eval.Integer,
) error {
	leftValue := left.Value
	rightValue := right.Value

	var result int64

	switch op {
	case opcode.OpAdd:
		result = leftValue + rightValue
	case opcode.OpSub:
		result = leftValue - rightValue
	case opcode.OpMul:
		result = leftValue * rightValue
	case opcode.OpDiv:
		result = leftValue / rightValue
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	return vm.push(eval.NewInteger(result))
}

func (vm *VM) executeComparison(op opcode.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	// Handle integer comparisons
	leftInt, leftOk := left.(*eval.Integer)
	rightInt, rightOk := right.(*eval.Integer)

	if leftOk && rightOk {
		return vm.executeIntegerComparison(op, leftInt, rightInt)
	}

	// Handle boolean comparisons
	switch op {
	case opcode.OpEqual:
		return vm.push(nativeBoolToBooleanObject(right == left))
	case opcode.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(right != left))
	default:
		return fmt.Errorf("unknown operator: %d (%s %s)",
			op, left.Kind(), right.Kind())
	}
}

func (vm *VM) executeIntegerComparison(
	op opcode.Opcode,
	left, right *eval.Integer,
) error {
	leftValue := left.Value
	rightValue := right.Value

	switch op {
	case opcode.OpEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue == leftValue))
	case opcode.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue != leftValue))
	case opcode.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(leftValue > rightValue))
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}
}

func nativeBoolToBooleanObject(input bool) *eval.Boolean {
	if input {
		return eval.TRUE
	}
	return eval.FALSE
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.pop()

	intOperand, ok := operand.(*eval.Integer)
	if !ok {
		return fmt.Errorf("unsupported type for negation: %s", operand.Kind())
	}

	return vm.push(eval.NewInteger(-intOperand.Value))
}

func (vm *VM) executeBangOperator() error {
	operand := vm.pop()

	switch operand {
	case eval.TRUE:
		return vm.push(eval.FALSE)
	case eval.FALSE:
		return vm.push(eval.TRUE)
	case eval.NULL:
		return vm.push(eval.TRUE)
	default:
		return vm.push(eval.FALSE)
	}
}
func (vm *VM) executeIndexExpression(left, index eval.Object) error {
	switch {
	case left.Kind() == eval.KindModule && index.Kind() == eval.KindString:
		module := left.(*eval.Module)
		key := index.(*eval.String).Value
		obj, ok := module.Env.Get(key)
		if !ok {
			return fmt.Errorf("property %s not found in module %s", key, module.Name)
		}
		if obj == nil {
			return fmt.Errorf("property %s in module %s is nil!", key, module.Name)
		}
		return vm.push(obj)
	default:
		return fmt.Errorf("index operator not supported: %s[%s]", left.Kind(), index.Kind())
	}
}
func isTruthy(obj eval.Object) bool {
	switch obj {
	case eval.NULL:
		return false
	case eval.TRUE:
		return true
	case eval.FALSE:
		return false
	default:
		return true
	}
}
