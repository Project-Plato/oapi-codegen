package runtime

import (
	"errors"
	"fmt"
	"reflect"
)

// https://gist.github.com/hvoecking/10772475

func Translate(obj interface{}) {
	// Wrap the original in a reflect.Value
	original := reflect.ValueOf(obj)

	if original.Kind() != reflect.Ptr {
		panic("must be pointer")
	}

	if original.IsNil() {
		panic("must be not nil")
	}

	copy := reflect.New(original.Type()).Elem()
	err := translateRecursive(copy, original)
	if err != nil {
		panic("translate failed, err: " + err.Error())
	}
	original.Elem().Set(copy.Elem())
}

// Recursively translates the structure, e.g. by converting nil slices to 0-sized slices.
func translateRecursive(copy, original reflect.Value) (err error) {
	switch original.Kind() {
	// The first cases handle nested structures and translate them recursively

	// If it is a pointer we need to unwrap and call once again
	case reflect.Ptr:
		// To get the actual value of the original we have to call Elem()
		// At the same time this unwraps the pointer so we don't end up in
		// an infinite recursion
		originalValue := original.Elem()
		// Check if the pointer is nil
		if !originalValue.IsValid() {
			return nil
		}
		// Allocate a new object and set the pointer to it
		copy.Set(reflect.New(originalValue.Type()))
		if translateRecursive(copy.Elem(), originalValue) != nil {
			copy.Set(original)
		}

		// If it is an interface (which is very similar to a pointer), do basically the
		// same as for the pointer. Though a pointer is not the same as an interface so
		// note that we have to call Elem() after creating a new object because otherwise
		// we would end up with an actual pointer

	case reflect.Interface:
		// Get rid of the wrapping interface
		originalValue := original.Elem()
		// Create a new object. Now new gives us a pointer, but we want the value it
		// points to, so we have to call Elem() to unwrap it
		copyValue := reflect.New(originalValue.Type()).Elem()
		err := translateRecursive(copyValue, originalValue)
		if err != nil {
			panic(fmt.Sprintf("can not translate interface, %v %v to %v %v, %v", originalValue.Kind(), originalValue.Type(), copyValue.Kind(), copyValue.Type(), err))
		}
		copy.Set(copyValue)
		//fmt.Println("interface", lapsus.Dump(copy.Interface()))

		// If it is a struct we translate each field
	case reflect.Struct:
		for i := 0; i < original.NumField(); i += 1 {
			//fmt.Println("struct", original.Type().Name(), "field", original.Type().Field(i).Name)
			if translateRecursive(copy.Field(i), original.Field(i)) != nil {
				if copy.Field(i).CanSet() {
					//fmt.Println("struct", original.Type().Name(), "field", original.Type().Field(i).Name, "unchanged")
					copy.Field(i).Set(original.Field(i))
				} else {
					//fmt.Println("struct", original.Type().Name(), "field", original.Type().Field(i).Name, "failed")
					return errors.New("cannot set: " + copy.Field(i).Type().String())
				}
			}
		}

		// If it is a slice we create a new slice and translate each element
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		//fmt.Println("array")
		copy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		if copy.IsNil() {
			panic("impossible")
		}
		for i := 0; i < original.Len(); i += 1 {
			if translateRecursive(copy.Index(i), original.Index(i)) != nil {
				//fmt.Println("array", i, "unchanged")
				copy.Index(i).Set(original.Index(i))
			}
		}
		//fmt.Println("array", lapsus.Dump(copy.Interface()))

		// If it is a map we create a new map and translate each value
	case reflect.Map:
		copy.Set(reflect.MakeMap(original.Type()))
		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			// New gives us a pointer, but again we want the value
			copyValue := reflect.New(originalValue.Type()).Elem()
			if translateRecursive(copyValue, originalValue) == nil {
				copy.SetMapIndex(key, copyValue)
			} else {
				copy.SetMapIndex(key, originalValue)
			}
		}
	default:
		if copy.CanSet() {
			copy.Set(original)
			return nil
		} else {
			return errors.New("cannot set: " + copy.Type().String())
		}
	}
	return nil
}
