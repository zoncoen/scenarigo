package template

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/zoncoen/scenarigo/internal/reflectutil"
)

var (
	typeInt64   = reflect.TypeOf(int64(0))
	typeUint64  = reflect.TypeOf(uint64(0))
	typeFloat64 = reflect.TypeOf(float64(0))
	typeBool    = reflect.TypeOf(false)
	typeString  = reflect.TypeOf("")
)

var defaultFuncs = map[string]interface{}{
	// type conversion
	"int":    convertToInt,
	"uint":   convertToUint,
	"float":  convertToFloat,
	"bool":   convertToBool,
	"string": convertToString,
}

func convertToInt(in interface{}) (int64, error) {
	if in == nil {
		return 0, fmt.Errorf("can't convert nil to int")
	}
	v := reflectutil.Elem(reflect.ValueOf(in))
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, ok := v.Convert(typeInt64).Interface().(int64)
		if ok {
			return i, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, ok := v.Convert(typeUint64).Interface().(uint64)
		if ok {
			if i > uint64(math.MaxInt64) {
				return 0, fmt.Errorf("%d overflows int", i)
			}
			return int64(i), nil
		}
	case reflect.Float32, reflect.Float64:
		f, ok := v.Convert(typeFloat64).Interface().(float64)
		if ok {
			return int64(f), nil
		}
	case reflect.String:
		s, ok := v.Convert(typeString).Interface().(string)
		if ok {
			i, err := strconv.ParseInt(s, 0, 64)
			if err != nil {
				return 0, err
			}
			return i, nil
		}
	case reflect.Invalid:
		return 0, fmt.Errorf("can't convert %#v to int", in)
	default:
		return 0, fmt.Errorf("can't convert %T to int", in)
	}
	return 0, fmt.Errorf("can't convert %T to int", in)
}

func convertToUint(in interface{}) (uint64, error) {
	if in == nil {
		return 0, fmt.Errorf("can't convert nil to uint")
	}
	v := reflectutil.Elem(reflect.ValueOf(in))
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, ok := v.Convert(typeInt64).Interface().(int64)
		if ok {
			if i < 0 {
				return 0, fmt.Errorf("can't convert %d to uint", i)
			}
			return uint64(i), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, ok := v.Convert(typeUint64).Interface().(uint64)
		if ok {
			return i, nil
		}
	case reflect.Float32, reflect.Float64:
		f, ok := v.Convert(typeFloat64).Interface().(float64)
		if ok {
			return uint64(f), nil
		}
	case reflect.String:
		s, ok := v.Convert(typeString).Interface().(string)
		if ok {
			i, err := strconv.ParseUint(s, 0, 64)
			if err != nil {
				return 0, err
			}
			return i, nil
		}
	case reflect.Invalid:
		return 0, fmt.Errorf("can't convert %#v to uint", in)
	default:
		return 0, fmt.Errorf("can't convert %T to uint", in)
	}
	return 0, fmt.Errorf("can't convert %T to uint", in)
}

func convertToFloat(in interface{}) (float64, error) {
	if in == nil {
		return 0, fmt.Errorf("can't convert nil to float")
	}
	v := reflectutil.Elem(reflect.ValueOf(in))
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, ok := v.Convert(typeInt64).Interface().(int64)
		if ok {
			return float64(i), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, ok := v.Convert(typeUint64).Interface().(uint64)
		if ok {
			return float64(i), nil
		}
	case reflect.Float32, reflect.Float64:
		f, ok := v.Convert(typeFloat64).Interface().(float64)
		if ok {
			return f, nil
		}
	case reflect.String:
		s, ok := v.Convert(typeString).Interface().(string)
		if ok {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return 0, err
			}
			return f, nil
		}
	case reflect.Invalid:
		return 0, fmt.Errorf("can't convert %#v to float", in)
	default:
		return 0, fmt.Errorf("can't convert %T to float", in)
	}
	return 0, fmt.Errorf("can't convert %T to float", in)
}

func convertToBool(in interface{}) (bool, error) {
	if in == nil {
		return false, fmt.Errorf("can't convert nil to bool")
	}
	v := reflectutil.Elem(reflect.ValueOf(in))
	switch v.Kind() {
	case reflect.Bool:
		if b, ok := v.Interface().(bool); ok {
			return b, nil
		}
	case reflect.Invalid:
		return false, fmt.Errorf("can't convert %#v to bool", in)
	default:
		return false, fmt.Errorf("can't convert %T to bool", in)
	}
	return false, fmt.Errorf("can't convert %T to bool", in)
}

func convertToString(in interface{}) (string, error) {
	if in == nil {
		return "", fmt.Errorf("can't convert nil to string")
	}
	v := reflectutil.Elem(reflect.ValueOf(in))
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, ok := v.Convert(typeInt64).Interface().(int64)
		if ok {
			return strconv.FormatInt(i, 10), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, ok := v.Convert(typeUint64).Interface().(uint64)
		if ok {
			return strconv.FormatUint(i, 10), nil
		}
	case reflect.Float32, reflect.Float64:
		f, ok := v.Convert(typeFloat64).Interface().(float64)
		if ok {
			return strconv.FormatFloat(f, 'f', -1, 64), nil
		}
	case reflect.String:
		s, ok := v.Convert(typeString).Interface().(string)
		if ok {
			return s, nil
		}
	case reflect.Invalid:
		return "", fmt.Errorf("can't convert %#v to string", in)
	default:
		return "", fmt.Errorf("can't convert %T to string", in)
	}
	return "", fmt.Errorf("can't convert %T to string", in)
}
