package gopy

import (
	"encoding/binary"
	"github.com/vmihailenco/msgpack/v5"
)

// Extension types for different array types and dimensions
const (
	ExtFloat32    = 11
	ExtFloat32_2D = 12
	ExtFloat32_3D = 13

	ExtFloat64    = 21
	ExtFloat64_2D = 22
	ExtFloat64_3D = 23

	ExtInt16    = 31
	ExtInt16_2D = 32
	ExtInt16_3D = 33

	ExtInt32    = 41
	ExtInt32_2D = 42
	ExtInt32_3D = 43

	ExtInt64    = 51
	ExtInt64_2D = 52
	ExtInt64_3D = 53
)

// Register all custom types with the MessagePack encoder/decoder
func RegisterTypes() {
	// Float32 types
	//msgpack.RegisterExt(ExtFloat32, (*Float32_Array)(nil))
	//msgpack.RegisterExt(ExtFloat32_2D, (*Float32_2DArray)(nil))
	//msgpack.RegisterExt(ExtFloat32_3D, (*Float32_3DArray)(nil))
	//
	//// Float64 types
	//msgpack.RegisterExt(ExtFloat64, (*Float64_Array)(nil))
	//msgpack.RegisterExt(ExtFloat64_2D, (*Float64_2DArray)(nil))
	//msgpack.RegisterExt(ExtFloat64_3D, (*Float64_3DArray)(nil))

	// Int16 types
	msgpack.RegisterExt(ExtInt16, (*Int16_Array)(nil))
	msgpack.RegisterExt(ExtInt16_2D, (*Int16_2DArray)(nil))
	msgpack.RegisterExt(ExtInt16_3D, (*Int16_3DArray)(nil))

	// Int32 types
	msgpack.RegisterExt(ExtInt32, (*Int32_Array)(nil))
	msgpack.RegisterExt(ExtInt32_2D, (*Int32_2DArray)(nil))
	msgpack.RegisterExt(ExtInt32_3D, (*Int32_3DArray)(nil))

	// Int64 types
	msgpack.RegisterExt(ExtInt64, (*Int64_Array)(nil))
	msgpack.RegisterExt(ExtInt64_2D, (*Int64_2DArray)(nil))
	msgpack.RegisterExt(ExtInt64_3D, (*Int64_3DArray)(nil))
}

// ------------------- Float32 types -------------------

type Float32_Array []float32
type Float32_2DArray [][]float32
type Float32_3DArray [][][]float32

type Float64_Array []float64
type Float64_2DArray [][]float64
type Float64_3DArray [][][]float64

type Int16_Array []int16
type Int16_2DArray [][]int16
type Int16_3DArray [][][]int16

type Int32_Array []int32
type Int32_2DArray [][]int32
type Int32_3DArray [][][]int32

type Int64_Array []int64
type Int64_2DArray [][]int64
type Int64_3DArray [][][]int64

// ------------------- Float32 types -------------------

func (arr Float32_Array) MarshalMsgpack() ([]byte, error) {
	l := uint32(len(arr) * 4)
	b := make([]byte, 6+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], l)
	b[5] = byte(ExtFloat32)
	float32ToBytes1D(arr, b[6:])
	return b, nil
}

func (arr *Float32_Array) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	*arr = make(Float32_Array, len(data)/4)
	bytesToFloat321D(data, *arr)
	return nil
}

func (arr Float32_2DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	if d1 > 0 {
		d2 = len(arr[0])
	}
	l := d1 * d2 * 4
	b := make([]byte, 6+8+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(8+l))
	b[5] = byte(ExtFloat32_2D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	float32ToBytes2D(arr, b[14:])
	return b, nil
}

func (arr *Float32_2DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	*arr = make(Float32_2DArray, d1)
	data = data[8:]
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([]float32, d2)
		bytesToFloat321D(data[i*d2*4:i*d2*4+d2*4], (*arr)[i])
	}
	return nil
}

func (arr Float32_3DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	var d3 int
	if d1 > 0 {
		d2 = len(arr[0])
		if d2 > 0 {
			d3 = len(arr[0][0])
		}
	}
	l := d1 * d2 * d3 * 4
	b := make([]byte, 18+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(l+12))
	b[5] = byte(ExtFloat32_3D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	binary.LittleEndian.PutUint32(b[14:18], uint32(d3))
	float32ToBytes3D(arr, b[18:])
	return b, nil
}

func (arr *Float32_3DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	d3 := int(binary.LittleEndian.Uint32(data[8:12]))
	*arr = make(Float32_3DArray, d1)
	data = data[12:]
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([][]float32, d2)
		for j := 0; j < d2; j++ {
			(*arr)[i][j] = make([]float32, d3)
			bytesToFloat321D(data[i*d2*d3*4+j*d3*4:i*d2*d3*4+j*d3*4+d3*4], (*arr)[i][j])
		}
	}
	return nil
}

// ------------------- Float64 types -------------------

func (arr Float64_Array) MarshalMsgpack() ([]byte, error) {
	l := uint64(len(arr) * 8)
	b := make([]byte, 6+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(l))
	b[5] = byte(ExtFloat64)
	float64ToBytes1D(arr, b[6:])
	return b, nil
}

func (arr *Float64_Array) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	*arr = make(Float64_Array, len(data)/8)
	bytesToFloat641D(data, *arr)
	return nil
}

func (arr Float64_2DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	if d1 > 0 {
		d2 = len(arr[0])
	}
	l := d1 * d2 * 8
	b := make([]byte, 6+8+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(8+l))
	b[5] = byte(ExtFloat64_2D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	float64ToBytes2D(arr, b[14:])
	return b, nil
}

func (arr *Float64_2DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	*arr = make(Float64_2DArray, d1)
	data = data[8:]
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([]float64, d2)
		bytesToFloat641D(data[i*d2*8:i*d2*8+d2*8], (*arr)[i])
	}
	return nil
}

func (arr Float64_3DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	var d3 int
	if d1 > 0 {
		d2 = len(arr[0])
		if d2 > 0 {
			d3 = len(arr[0][0])
		}
	}
	l := d1 * d2 * d3 * 8
	b := make([]byte, 18+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(l+12))
	b[5] = byte(ExtFloat64_3D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	binary.LittleEndian.PutUint32(b[14:18], uint32(d3))
	float64ToBytes3D(arr, b[18:])
	return b, nil
}

func (arr *Float64_3DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	d3 := int(binary.LittleEndian.Uint32(data[8:12]))
	*arr = make(Float64_3DArray, d1)
	data = data[12:]
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([][]float64, d2)
		for j := 0; j < d2; j++ {
			(*arr)[i][j] = make([]float64, d3)
			bytesToFloat641D(data[i*d2*d3*8+j*d3*8:i*d2*d3*8+j*d3*8+d3*8], (*arr)[i][j])
		}
	}
	return nil
}

// ------------------- Int16 types -------------------

func (arr Int16_Array) MarshalMsgpack() ([]byte, error) {
	l := uint32(len(arr) * 2)
	b := make([]byte, 6+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], l)
	b[5] = byte(ExtInt16)
	int16ToBytes1D(arr, b[6:])
	return b, nil
}

func (arr *Int16_Array) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	*arr = make(Int16_Array, len(data)/2)
	bytesToInt161D(data, *arr)
	return nil
}

func (arr Int16_2DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	if d1 > 0 {
		d2 = len(arr[0])
	}
	l := d1 * d2 * 2
	b := make([]byte, 6+8+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(8+l))
	b[5] = byte(ExtInt16_2D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	int16ToBytes2D(arr, b[14:])
	return b, nil
}

func (arr *Int16_2DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	*arr = make(Int16_2DArray, d1)
	data = data[8:]
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([]int16, d2)
		bytesToInt161D(data[i*d2*2:i*d2*2+d2*2], (*arr)[i])
	}
	return nil
}

func (arr Int16_3DArray) MarshalMsgpack() ([]byte, error) {
	panic("not implemented")
}

func (arr *Int16_3DArray) UnmarshalMsgpack(data []byte) error {
	panic("not implemented")
}

// ------------------- Int32 types -------------------

func (arr Int32_Array) MarshalMsgpack() ([]byte, error) {
	l := uint32(len(arr) * 4)
	b := make([]byte, 6+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], l)
	b[5] = byte(ExtInt32)
	int32ToBytes1D(arr, b[6:])
	return b, nil
}

func (arr *Int32_Array) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	*arr = make(Int32_Array, len(data)/4)
	bytesToInt321D(data, *arr)
	return nil
}

func (arr Int32_2DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	if d1 > 0 {
		d2 = len(arr[0])
	}
	l := d1 * d2 * 4
	b := make([]byte, 6+8+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(8+l))
	b[5] = byte(ExtInt32_2D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	int32ToBytes2D(arr, b[14:])
	return b, nil
}

func (arr *Int32_2DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	*arr = make(Int32_2DArray, d1)
	data = data[8:]
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([]int32, d2)
		bytesToInt321D(data[i*d2*4:i*d2*4+d2*4], (*arr)[i])
	}
	return nil
}

func (arr Int32_3DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	var d3 int
	if d1 > 0 {
		d2 = len(arr[0])
		if d2 > 0 {
			d3 = len(arr[0][0])
		}
	}
	l := d1 * d2 * d3 * 4
	b := make([]byte, 18+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(l+12))
	b[5] = byte(ExtInt32_3D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	binary.LittleEndian.PutUint32(b[14:18], uint32(d3))
	int32ToBytes3D(arr, b[18:])
	return b, nil
}

func (arr *Int32_3DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	d3 := int(binary.LittleEndian.Uint32(data[8:12]))
	*arr = make(Int32_3DArray, d1)
	data = data[12:] // Adjust offset to skip the dimension headers
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([][]int32, d2)
		for j := 0; j < d2; j++ {
			(*arr)[i][j] = make([]int32, d3)
			bytesToInt321D(data[i*d2*d3*4+j*d3*4:i*d2*d3*4+j*d3*4+d3*4], (*arr)[i][j])
		}
	}
	return nil
}

// ------------------- Int64 types -------------------

func (arr Int64_Array) MarshalMsgpack() ([]byte, error) {
	l := uint32(len(arr) * 8)
	b := make([]byte, 6+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], l)
	b[5] = byte(ExtInt64)
	int64ToBytes1D(arr, b[6:])
	return b, nil
}

func (arr *Int64_Array) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	*arr = make(Int64_Array, len(data)/8)
	bytesToInt641D(data, *arr)
	return nil
}

func (arr Int64_2DArray) MarshalMsgpack() ([]byte, error) {
	var d1 = len(arr)
	var d2 int
	if d1 > 0 {
		d2 = len(arr[0])
	}
	l := d1 * d2 * 8
	b := make([]byte, 6+8+l)
	b[0] = 0xC9
	binary.BigEndian.PutUint32(b[1:5], uint32(8+l))
	b[5] = byte(ExtInt64_2D)
	binary.LittleEndian.PutUint32(b[6:10], uint32(d1))
	binary.LittleEndian.PutUint32(b[10:14], uint32(d2))
	int64ToBytes2D(arr, b[14:])
	return b, nil
}

func (arr *Int64_2DArray) UnmarshalMsgpack(data []byte) error {
	data, err := stripMsgPackHeader(data)
	if err != nil {
		return err
	}
	d1 := int(binary.LittleEndian.Uint32(data[:4]))
	d2 := int(binary.LittleEndian.Uint32(data[4:8]))
	*arr = make(Int64_2DArray, d1)
	data = data[8:]
	for i := 0; i < d1; i++ {
		(*arr)[i] = make([]int64, d2)
		bytesToInt641D(data[i*d2*8:i*d2*8+d2*8], (*arr)[i])
	}
	return nil
}

func (arr Int64_3DArray) MarshalMsgpack() ([]byte, error) {
	panic("not implemented")
}

func (arr *Int64_3DArray) UnmarshalMsgpack(data []byte) error {
	panic("not implemented")
}
