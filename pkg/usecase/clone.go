package usecase

import (
	"go/ast"
	"reflect"
)

func clone(value reflect.Value) reflect.Value {
	adjustValue := func(ret reflect.Value) reflect.Value {
		switch value.Kind() {
		case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Array:
			return ret
		default:
			return ret.Elem()
		}
	}

	src := value
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.New(value.Type()).Elem()
		}
		src = value.Elem()
	}

	var dst reflect.Value

	switch src.Kind() {
	case reflect.String:
		dst = reflect.New(src.Type())
		dst.Elem().SetString(value.String())

	case reflect.Struct:
		dst = reflect.New(src.Type())
		t := src.Type()

		for i := 0; i < t.NumField(); i++ {
			fv := src.Field(i)

			objType := reflect.TypeOf(&ast.Object{})

			if fv.Type() == objType {
				empty := reflect.New(fv.Type())
				dst.Elem().Field(i).Set(empty.Elem())
				continue
			}

			if !fv.CanInterface() {
				continue
			}

			dst.Elem().Field(i).Set(clone(fv))
		}

	case reflect.Map:
		dst = reflect.MakeMap(src.Type())
		keys := src.MapKeys()
		for i := 0; i < src.Len(); i++ {
			mValue := src.MapIndex(keys[i])
			dst.SetMapIndex(keys[i], clone(mValue))
		}

	case reflect.Array, reflect.Slice:
		dst = reflect.MakeSlice(src.Type(), src.Len(), src.Cap())
		for i := 0; i < src.Len(); i++ {
			srcValue := src.Index(i)
			newValue := clone(srcValue)
			dst.Index(i).Set(newValue)
		}

	case reflect.Interface:
		dst = reflect.New(src.Type())
		if !src.IsNil() {
			dst.Elem().Set(clone(src.Elem()))
		}

	default:
		dst = reflect.New(src.Type())
		dst.Elem().Set(src)
	}

	return adjustValue(dst)
}
