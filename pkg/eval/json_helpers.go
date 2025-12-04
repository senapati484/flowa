package eval

import (
	"fmt"
)

// Helper to convert Flowa objects to native Go types for JSON marshaling
func FlowaToNative(obj Object) interface{} {
	switch obj := obj.(type) {
	case *Integer:
		return obj.Value
	case *String:
		return obj.Value
	case *Boolean:
		return obj.Value
	case *Null:
		return nil
	case *Array:
		result := make([]interface{}, 0, len(obj.Elements))
		for _, elem := range obj.Elements {
			result = append(result, FlowaToNative(elem))
		}
		return result
	case *Map:
		result := make(map[string]interface{})
		for k, v := range obj.Pairs {
			keyStr := k.Inspect()
			if s, ok := k.(*String); ok {
				keyStr = s.Value
			}
			result[keyStr] = FlowaToNative(v)
		}
		return result
	case *StructInstance:
		result := make(map[string]interface{})
		for k, v := range obj.Fields {
			result[k] = FlowaToNative(v)
		}
		return result
	default:
		return obj.Inspect()
	}
}

// Helper to convert native Go types to Flowa objects after JSON unmarshaling
func NativeToFlowa(val interface{}) Object {
	switch v := val.(type) {
	case nil:
		return NULL
	case bool:
		if v {
			return TRUE
		}
		return FALSE
	case float64:
		return &Integer{Value: int64(v)}
	case int:
		return &Integer{Value: int64(v)}
	case int64:
		return &Integer{Value: v}
	case string:
		return &String{Value: v}
	case []interface{}:
		elements := make([]Object, 0, len(v))
		for _, e := range v {
			elements = append(elements, NativeToFlowa(e))
		}
		return &Array{Elements: elements}
	case map[string]interface{}:
		pairs := make(map[Object]Object)
		for k, val := range v {
			pairs[&String{Value: k}] = NativeToFlowa(val)
		}
		return &Map{Pairs: pairs}
	default:
		return &String{Value: fmt.Sprintf("%v", v)}
	}
}
