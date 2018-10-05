package filter

import (
	"errors"
	"reflect"
)

// errors
var (
	ErrNilArguments            = errors.New("src and dst must not be nil")
	ErrNotSupported            = errors.New("only structs and maps are supported")
	ErrDifferentArgumentsTypes = errors.New("src and dst must be of same type")
)

type visit struct {
	ptr  uintptr
	typ  reflect.Type
	next *visit
}

// MergeWithOverwrite is
func MergeWithOverwrite(dst, src interface{}) error {
	return merge(dst, src, true)
}

func merge(dst, src interface{}, overwrite bool) error {
	var (
		vDst, vSrc reflect.Value
		err        error
	)

	if vDst, vSrc, err = resolveValues(dst, src); err != nil {
		return err
	}

	if vDst.Type() != vSrc.Type() {
		return ErrDifferentArgumentsTypes
	}

	return deepMerge(vDst, vSrc, make(map[uintptr]*visit), 0, overwrite)
}

func deepMerge(dst, src reflect.Value, visited map[uintptr]*visit, depth int, overwrite bool) (err error) {
	if !src.IsValid() {
		return
	}

	if dst.CanAddr() {
		addr := dst.UnsafeAddr()
		h := 17 * addr
		seen := visited[h]
		typ := dst.Type()
		for p := seen; p != nil; p = p.next {
			if p.ptr == addr && p.typ == typ {
				return nil
			}
		}

		visited[h] = &visit{addr, typ, seen}
	}

	switch dst.Kind() {
	case reflect.Struct:
		for i, n := 0, dst.NumField(); i < n; i++ {
			if err = deepMerge(dst.Field(i), src.Field(i), visited, depth+1, overwrite); err != nil {
				return
			}
		}
	case reflect.Map:
		for _, key := range src.MapKeys() {
			srcElement := src.MapIndex(key)
			if !srcElement.IsValid() {
				continue
			}
			dstElement := dst.MapIndex(key)
			switch srcElement.Kind() {
			case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
				if srcElement.IsNil() {
					continue
				}
				fallthrough
			default:
				if !srcElement.CanInterface() {
					continue
				}
				switch reflect.TypeOf(srcElement.Interface()).Kind() {
				case reflect.Struct:
					fallthrough
				case reflect.Ptr:
					fallthrough
				case reflect.Map:
					if err = deepMerge(dstElement, srcElement, visited, depth+1, overwrite); err != nil {
						return
					}
				}
			}
			if !isEmptyValue(srcElement) && (overwrite || (!dstElement.IsValid() || isEmptyValue(dst))) {
				if dst.IsNil() {
					dst.Set(reflect.MakeMap(dst.Type()))
				}
				dst.SetMapIndex(key, srcElement)
			}
		}
	case reflect.Ptr:
		fallthrough
	case reflect.Interface:
		if src.IsNil() {
			break
		} else if dst.IsNil() || overwrite {
			if dst.CanSet() && (overwrite || isEmptyValue(dst)) {
				dst.Set(src)
			}
		} else if err = deepMerge(dst.Elem(), src.Elem(), visited, depth+1, overwrite); err != nil {
			return
		}
	default:
		if dst.CanSet() && !isEmptyValue(src) && (overwrite || isEmptyValue(dst)) {
			dst.Set(src)
		}
	}
	return
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func resolveValues(dst, src interface{}) (vDst, vSrc reflect.Value, err error) {
	if dst == nil || src == nil {
		err = ErrNilArguments
		return
	}

	vDst = reflect.ValueOf(dst).Elem()
	if vDst.Kind() != reflect.Struct && vDst.Kind() != reflect.Map {
		err = ErrNotSupported
		return
	}

	vSrc = reflect.ValueOf(src)
	if vSrc.Kind() == reflect.Ptr {
		vSrc = vSrc.Elem()
	}
	return
}
