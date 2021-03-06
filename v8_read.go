package v8

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ReadInto reads a v8 Value into a Go variable passed by reference,
// and returns nil upon success and an error in case type cast is
// not possible returns.
func ReadInto(varPtr interface{}, value *Value, maxDepth int) error {
	path := make([]string, 0, maxDepth)
	return readInto(varPtr, value, path, maxDepth)
}

func readInto(dst interface{}, value *Value, path []string, maxDepth int) (err error) {

	if len(path) > maxDepth {
		return fmt.Errorf("max depth of %d exceeded", maxDepth)
	}

	if reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return errors.New("dst is not a pointer")
	}

	// get reflect Value
	dstPtrVal := reflect.ValueOf(dst)

	// get pointer type
	dstPtrType := dstPtrVal.Type()

	// get destination type
	dstType := dstPtrType.Elem()

	// get value that the pointer points to
	dstValue := reflect.Indirect(dstPtrVal)

	if value.IsKind(KindUndefined) || value.IsKind(KindNull) {
		dstValue.Set(reflect.Zero(dstType))
		return nil
	}

	switch dstType.Kind() {
	case reflect.Invalid:
		return getReadIntoError("invalid variable kind", path)
	case reflect.Bool:
		dstValue.Set(reflect.ValueOf(value.Bool()))
	case reflect.Int:
		dstValue.Set(reflect.ValueOf(int(value.Int64())))
	case reflect.Int8:
		dstValue.Set(reflect.ValueOf(int8(value.Int64())))
	case reflect.Int16:
		dstValue.Set(reflect.ValueOf(int16(value.Int64())))
	case reflect.Int32:
		dstValue.Set(reflect.ValueOf(int32(value.Int64())))
	case reflect.Int64:
		dstValue.Set(reflect.ValueOf(int64(value.Int64())))
	case reflect.Uint:
		dstValue.Set(reflect.ValueOf(uint(value.Int64())))
	case reflect.Uint8:
		dstValue.Set(reflect.ValueOf(uint8(value.Int64())))
	case reflect.Uint16:
		dstValue.Set(reflect.ValueOf(uint16(value.Int64())))
	case reflect.Uint32:
		dstValue.Set(reflect.ValueOf(uint32(value.Int64())))
	case reflect.Uint64:
		dstValue.Set(reflect.ValueOf(uint64(value.Int64())))
	case reflect.Uintptr:
		return getReadIntoError("uintptr not supported", path)
	case reflect.Float32:
		dstValue.Set(reflect.ValueOf(float32(value.Float64())))
	case reflect.Float64:
		dstValue.Set(reflect.ValueOf(float64(value.Float64())))
	case reflect.Complex64:
		fallthrough
	case reflect.Complex128:
		return getReadIntoError("complex not supported", path)
	case reflect.Array:
	case reflect.Chan:
		return getReadIntoError("chan not supported", path)
	case reflect.Func:
		return getReadIntoError("func not supported", path)
	case reflect.Interface:
		return getReadIntoError("interface not supported", path)
	case reflect.Map:
		if !value.IsKind(KindObject) {
			return getReadIntoError("value to be read into a map is not an object", path)
		}
		for _, mapKey := range dstValue.MapKeys() {
			objVal, err := value.Get(mapKey.String())
			if err != nil {
				return err
			}
			err = readInto(dstValue.MapIndex(mapKey).Addr().Interface(), objVal, append(path, mapKey.String()), maxDepth)
			if err != nil {
				return err
			}
		}
	case reflect.Ptr:
		return getReadIntoError("pointer not supported", path)
	case reflect.Slice:
		if !value.IsKind(KindArray) && !value.IsKind(KindObject) {
			return getReadIntoError("value to be read into a slice is not an array or object", path)
		}

		// get the length of the V8 Array
		lengthVal, err := value.Get("length")
		if err != nil {
			return err
		}
		length := int(lengthVal.Int64())

		// Make Go slice
		newSlice := reflect.MakeSlice(reflect.SliceOf(dstType.Elem()), length, length)
		for i := 0; i < length; i++ {
			arrVal, err := value.GetIndex(i)
			err = readInto(newSlice.Index(i).Addr().Interface(), arrVal, append(path, string(i)), maxDepth)
			if err != nil {
				return err
			}
		}

		//JV added, panel.is bugfix
		dstValue.Set(newSlice)

	case reflect.String:
		dstValue.Set(reflect.ValueOf(value.String()))

	case reflect.Struct:
		if !value.IsKind(KindObject) {
			return getReadIntoError("value to be read into a map is not an object", path)
		}

		fieldCount := dstType.NumField()
		for i := 0; i < fieldCount; i++ {

			field := dstType.Field(i)

			jsonFieldName := field.Tag.Get("json")

			jsonOptIndex := strings.Index(jsonFieldName, ",")
			if jsonOptIndex >= 0 {
				jsonFieldName = jsonFieldName[0:jsonOptIndex]
			}

			if jsonFieldName == "" {
				jsonFieldName = field.Name
			}

			objVal, err := value.Get(jsonFieldName)
			if err != nil {
				return err
			}

			err = readInto(dstValue.Field(i).Addr().Interface(), objVal, append(path, field.Name), maxDepth)
			if err != nil {
				return err
			}
		}

	case reflect.UnsafePointer:
		return getReadIntoError("unsafe pointer not supported", path)
	default:
		return fmt.Errorf("unsupported variable kind: %d", dstType.Kind())
	}
	return nil
}

func getReadIntoError(msg string, path []string) error {
	if len(path) == 0 {
		return errors.New(msg)
	}
	return fmt.Errorf("error reading value into %s: %s", strings.Join(path, "."), msg)
}
