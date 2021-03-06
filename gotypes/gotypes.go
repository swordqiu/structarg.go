package gotypes

import (
    "fmt"
    "reflect"
    "strconv"
)


const (
    EMPTYSTR = ""
)

var (
    boolValue bool
    intValue int
    int8Value int8
    int16Value int16
    int32Value int32
    int64Value int64
    uintValue uint
    uint8Value uint8
    uint16Value uint16
    uint32Value uint32
    uint64Value uint64
    float32Value float32
    float64Value float64
    stringValue string
    boolSliceValue  []bool
    intSliceValue   []int
    int8SliceValue  []int8
    int16SliceValue []int16
    int32SliceValue  []int32
    int64SliceValue  []int64
    uintSliceValue   []uint
    uint8SliceValue  []uint8
    uint16SliceValue []uint16
    uint32SliceValue []uint32
    uint64SliceValue []uint64
    float32SliceValue []float32
    float64SliceValue []float64
    stringSliceValue  []string
)


var (
    BoolType = reflect.TypeOf(boolValue)
    IntType = reflect.TypeOf(intValue)
    Int8Type = reflect.TypeOf(int8Value)
    Int16Type = reflect.TypeOf(int16Value)
    Int32Type = reflect.TypeOf(int32Value)
    Int64Type = reflect.TypeOf(int64Value)
    UintType  = reflect.TypeOf(uintValue)
    Uint8Type = reflect.TypeOf(uint8Value)
    Uint16Type = reflect.TypeOf(uint16Value)
    Uint32Type = reflect.TypeOf(uint32Value)
    Uint64Type = reflect.TypeOf(uint64Value)
    Float32Type = reflect.TypeOf(float32Value)
    Float64Type = reflect.TypeOf(float64Value)
    StringType  = reflect.TypeOf(stringValue)
    BoolSliceType = reflect.TypeOf(boolSliceValue)
    IntSliceType  = reflect.TypeOf(intSliceValue)
    Int8SliceType = reflect.TypeOf(int8SliceValue)
    Int16SliceType = reflect.TypeOf(int16SliceValue)
    Int32SliceType = reflect.TypeOf(int32SliceValue)
    Int64SliceType = reflect.TypeOf(int64SliceValue)
    UintSliceType  = reflect.TypeOf(uintSliceValue)
    Uint8SliceType = reflect.TypeOf(uint8SliceValue)
    Uint16SliceType = reflect.TypeOf(uint16SliceValue)
    Uint32SliceType = reflect.TypeOf(uint32SliceValue)
    Uint64SliceType = reflect.TypeOf(uint64SliceValue)
    Float32SliceType = reflect.TypeOf(float32SliceValue)
    Float64SliceType = reflect.TypeOf(float64SliceValue)
    StringSliceType = reflect.TypeOf(stringSliceValue)
)


func ParseValue(val string, tp reflect.Type) (reflect.Value, error) {
    switch tp {
        case BoolType:
            val_bool, err := strconv.ParseBool(val)
            return reflect.ValueOf(val_bool), err
        case IntType, Int8Type, Int16Type, Int32Type, Int64Type:
            val_int, err := strconv.ParseInt(val, 10, 64)
            switch tp {
            case IntType:
                return reflect.ValueOf(int(val_int)), err
            case Int8Type:
                return reflect.ValueOf(int8(val_int)), err
            case Int16Type:
                return reflect.ValueOf(int16(val_int)), err
            case Int32Type:
                return reflect.ValueOf(int32(val_int)), err
            default:
                return reflect.ValueOf(val_int), err
            }
        case UintType, Uint8Type, Uint16Type, Uint32Type, Uint64Type:
            val_uint, err := strconv.ParseUint(val, 10, 64)
            switch tp {
            case UintType:
                return reflect.ValueOf(uint(val_uint)), err
            case Uint8Type:
                return reflect.ValueOf(uint8(val_uint)), err
            case Uint16Type:
                return reflect.ValueOf(uint16(val_uint)), err
            case Uint32Type:
                return reflect.ValueOf(uint32(val_uint)), err
            default:
                return reflect.ValueOf(val_uint), err
            }
        case Float32Type, Float64Type:
            val_float, err := strconv.ParseFloat(val, 64)
            if tp == Float32Type {
                return reflect.ValueOf(float32(val_float)), err
            }else {
                return reflect.ValueOf(val_float), err
            }
        case StringType:
            return reflect.ValueOf(val), nil
        default:
            return reflect.ValueOf(val), fmt.Errorf("Cannot parse %s to %s", val, tp)
    }
}


func SetValue(value reflect.Value, val string) error {
    if ! value.CanSet() {
        fmt.Errorf("Value is not settable")
    }
    switch value.Type() {
        case BoolType:
            val_bool, e := strconv.ParseBool(val)
            if e != nil {
                return e
            }
            value.SetBool(val_bool)
        case IntType, Int8Type, Int16Type, Int32Type, Int64Type:
            val_int, e := strconv.ParseInt(val, 10, 64)
            if e != nil {
                return e
            }
            value.SetInt(val_int)
        case UintType, Uint8Type, Uint16Type, Uint32Type, Uint64Type:
            val_uint, e := strconv.ParseUint(val, 10, 64)
            if e != nil {
                return e
            }
            value.SetUint(val_uint)
        case Float32Type, Float64Type:
            val_float, e := strconv.ParseFloat(val, 64)
            if e != nil {
                return e
            }
            value.SetFloat(val_float)
        case StringType:
            value.SetString(val)
        default:
            return fmt.Errorf("Unsupported type: %s", value.Type)
    }
    return nil
}


func AppendValues(value reflect.Value, vals ...string) error {
    var e error = nil
    for _, val := range vals {
        e = AppendValue(value, val)
        if e != nil {
            break
        }
    }
    return e
}


func SliceBaseType(tp reflect.Type) reflect.Type {
    switch tp {
        case BoolSliceType:
            return BoolType
        case IntSliceType:
            return IntType
        case Int8SliceType:
            return Int8Type
        case Int16SliceType:
            return Int16Type
        case Int32SliceType:
            return Int32Type
        case Int64SliceType:
            return Int64Type
        case UintSliceType:
            return UintType
        case Uint8SliceType:
            return Uint8Type
        case Uint16SliceType:
            return Uint16Type
        case Uint32SliceType:
            return Uint32Type
        case Uint64SliceType:
            return Uint64Type
        case Float32SliceType:
            return Float32Type
        case Float64SliceType:
            return Float64Type
        case StringSliceType:
            return StringType
        default:
            return nil
    }
}

func AppendValue(value reflect.Value, val string) error {
    tp := SliceBaseType(value.Type())
    if tp == nil {
        return fmt.Errorf("Cannot append to non-slice type")
    }
    val_raw, e := ParseValue(val, tp)
    if e != nil {
        return e
    }
    value.Set(reflect.Append(value, val_raw))
    return nil
}

func InCollection(obj interface{}, array interface{}) bool {
    var arrVal = reflect.ValueOf(array)
    var arrKind = arrVal.Type().Kind()
    var arrSet []reflect.Value
    if arrKind == reflect.Map {
        arrSet = arrVal.MapKeys()
    }else if arrKind == reflect.Array || arrKind == reflect.Slice {
        arrSet = make([]reflect.Value, 0)
        for i:= 0; i < arrVal.Len(); i ++ {
            arrSet = append(arrSet, arrVal.Index(i))
        }
    }else {
        return false
    }
    for _, arrObj := range arrSet {
        if reflect.DeepEqual(obj, arrObj.Interface()) {
            return true
        }
    }
    return false
}
