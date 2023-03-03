package xreflect

import (
	"reflect"
	"time"
)

var IntType = reflect.TypeOf(0)
var IntPtrType = reflect.PtrTo(IntType)
var Int8Type = reflect.TypeOf(int8(0))
var Int8PtrType = reflect.PtrTo(Int8Type)
var Int16Type = reflect.TypeOf(int16(0))
var Int16PtrType = reflect.PtrTo(Int16Type)
var Int32Type = reflect.TypeOf(int32(0))
var Int32PtrType = reflect.PtrTo(Int32Type)
var Int64Type = reflect.TypeOf(int64(0))
var Int64PtrType = reflect.PtrTo(Int64Type)

var UintType = reflect.TypeOf(uint(0))
var UintPtrType = reflect.PtrTo(UintType)
var Uint8Type = reflect.TypeOf(uint8(0))
var Uint8PtrType = reflect.PtrTo(Uint8Type)
var Uint16Type = reflect.TypeOf(uint16(0))
var Uint16PtrType = reflect.PtrTo(Uint16Type)
var Uint32Type = reflect.TypeOf(uint32(0))
var Uint32PtrType = reflect.PtrTo(Uint32Type)
var Uint64Type = reflect.TypeOf(uint64(0))
var Uint64PtrType = reflect.PtrTo(Uint64Type)

var Float32Type = reflect.TypeOf(float32(0.0))
var Float32PtrType = reflect.PtrTo(Float32Type)
var Float64Type = reflect.TypeOf(0.0)
var Float64PtrType = reflect.PtrTo(Float64Type)

var StringType = reflect.TypeOf("")
var StringPtrType = reflect.PtrTo(StringType)
var BoolType = reflect.TypeOf(false)
var BoolPtrType = reflect.PtrTo(BoolType)

var TimeType = reflect.TypeOf(time.Time{})
var TimePtrType = reflect.PtrTo(TimeType)
var InterfaceType = reflect.TypeOf(aStruct{}).Field(0).Type
var InterfacePtrType = reflect.PtrTo(InterfaceType)

var ErrorType = reflect.TypeOf(aStruct{}).Field(1).Type
var ErrorPtrType = reflect.PtrTo(ErrorType)

var BytesType = reflect.TypeOf([]byte{})
var BytesPtrType = reflect.PtrTo(BytesType)

type aStruct struct {
	ifaceField interface{}
	errField   error
}
