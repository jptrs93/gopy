package gopy

import (
	"context"
	"embed"
	"math/rand"
	"os/exec"
	"reflect"
	"testing"
)

//go:embed test-scripts/*
var scriptsFS embed.FS

type AddInput struct {
	A int `msgpack:"a,omitempty"`
	B int `msgpack:"b,omitempty"`
}

type AddResult struct {
	Result int `msgpack:"result,omitempty"`
}

type AddArraysInput struct {
	A Float64_Array `msgpack:"a,omitempty"`
	B Float64_Array `msgpack:"b,omitempty"`
}

type ArrayWrapperFloat64 struct {
	Arr2D Float64_2DArray `msgpack:"arr2D,omitempty"`
	Arr1D Float64_Array   `msgpack:"arr1D,omitempty"`
}

type ArrayWrapperFloat32 struct {
	Arr2D Float32_2DArray `msgpack:"arr2D,omitempty"`
	Arr1D Float32_Array   `msgpack:"arr1D,omitempty"`
}

func TestCall(t *testing.T) {
	pythonEnv, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("python3 not found: %v", err)
	}
	pp := NewPool(context.Background(), scriptsFS, pythonEnv, "test_script.py", 1)
	defer pp.Close()

	// build a test array
	columns := 10 + rand.Intn(20)
	var testArr Float64_2DArray
	var testArrFloat32 Float32_2DArray
	for i := 0; i < 10+rand.Intn(20); i++ {
		var row []float64
		var rowFloat32 []float32
		for c := 0; c < columns; c++ {
			row = append(row, rand.Float64())
			rowFloat32 = append(rowFloat32, rand.Float32())
		}
		testArr = append(testArr, row)
		testArrFloat32 = append(testArrFloat32, rowFloat32)
	}

	type testCase struct {
		name     string
		exec     func() (any, error)
		want     any
		wantType reflect.Type
		wantErr  bool
	}

	tests := []testCase{
		{
			name: "add with dict response",
			exec: func() (any, error) {
				return CallPool[AddResult](pp, "add", AddInput{5, 6})
			},
			want: AddResult{11},
		},
		{
			name: "add with scalar response",
			exec: func() (any, error) {
				return CallPool[int](pp, "add_scalar_output", AddInput{5, 6})
			},
			want: 11,
		},
		{
			name: "test int32 1D empty array ",
			exec: func() (any, error) {
				return CallPool[Int32_Array](pp, "identity", Int32_Array{})
			},
			want: Int32_Array{},
		},
		{
			name: "test int32 1D array ",
			exec: func() (any, error) {
				return CallPool[Int32_Array](pp, "identity", Int32_Array{1, 2, 3, 3, 4, 100})
			},
			want: Int32_Array{1, 2, 3, 3, 4, 100},
		},
		{
			name: "test int32 2D empty array ",
			exec: func() (any, error) {
				return CallPool[Int32_2DArray](pp, "identity", Int32_2DArray{})
			},
			want: Int32_2DArray{},
		},
		{
			name: "test int32 2D array ",
			exec: func() (any, error) {
				return CallPool[Int32_2DArray](pp, "identity", Int32_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}})
			},
			want: Int32_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}},
		},
		{
			name: "test int32 3D empty array ",
			exec: func() (any, error) {
				return CallPool[Int32_3DArray](pp, "identity", Int32_3DArray{})
			},
			want: Int32_3DArray{},
		},
		{
			name: "test int32 3D array ",
			exec: func() (any, error) {
				return CallPool[Int32_3DArray](pp, "identity", Int32_3DArray{{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}}, {{1, 8, 5, 4, 1, 100}, {1, 8, 5, 4, 1, 99}}})
			},
			want: Int32_3DArray{{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}}, {{1, 8, 5, 4, 1, 100}, {1, 8, 5, 4, 1, 99}}},
		},
		{
			name: "test int64 1D empty array ",
			exec: func() (any, error) {
				return CallPool[Int64_Array](pp, "identity", Int64_Array{})
			},
			want: Int64_Array{},
		},
		{
			name: "test int64 1D array ",
			exec: func() (any, error) {
				return CallPool[Int64_Array](pp, "identity", Int64_Array{1, 2, 3, 3, 4, 100})
			},
			want: Int64_Array{1, 2, 3, 3, 4, 100},
		},
		{
			name: "test int64 2D empty array ",
			exec: func() (any, error) {
				return CallPool[Int64_2DArray](pp, "identity", Int64_2DArray{{}, {}})
			},
			want: Int64_2DArray{{}, {}},
		},
		{
			name: "test int64 2D array ",
			exec: func() (any, error) {
				return CallPool[Int64_2DArray](pp, "identity", Int64_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}})
			},
			want: Int64_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}},
		},
		{
			name: "test int16 1D array ",
			exec: func() (any, error) {
				return CallPool[Int16_Array](pp, "identity", Int16_Array{1, 2, 3, 3, 4, 100})
			},
			want: Int16_Array{1, 2, 3, 3, 4, 100},
		},
		{
			name: "test int16 2D array ",
			exec: func() (any, error) {
				return CallPool[Int16_2DArray](pp, "identity", Int16_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}})
			},
			want: Int16_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}},
		},
		{
			name: "test float32 1D empty array ",
			exec: func() (any, error) {
				return CallPool[Float32_Array](pp, "identity", Float32_Array{})
			},
			want: Float32_Array{},
		},
		{
			name: "test float32 1D array ",
			exec: func() (any, error) {
				return CallPool[Float32_Array](pp, "identity", Float32_Array{1, 2, 3, 3, 4, 100})
			},
			want: Float32_Array{1, 2, 3, 3, 4, 100},
		},
		{
			name: "test float32 2D empty array ",
			exec: func() (any, error) {
				return CallPool[Float32_2DArray](pp, "identity", Float32_2DArray{{}, {}})
			},
			want: Float32_2DArray{{}, {}},
		},
		{
			name: "test float32 2D array ",
			exec: func() (any, error) {
				return CallPool[Float32_2DArray](pp, "identity", Float32_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}})
			},
			want: Float32_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}},
		},
		{
			name: "test float32 3D empty array ",
			exec: func() (any, error) {
				return CallPool[Float32_3DArray](pp, "identity", Float32_3DArray{})
			},
			want: Float32_3DArray{},
		},
		{
			name: "test float32 3D array ",
			exec: func() (any, error) {
				return CallPool[Float32_3DArray](pp, "identity", Float32_3DArray{{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}}, {{1, 8, 5, 4, 1, 100}, {1, 8, 5, 4, 1, 99}}})
			},
			want: Float32_3DArray{{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}}, {{1, 8, 5, 4, 1, 100}, {1, 8, 5, 4, 1, 99}}},
		},
		{
			name: "test float64 1D empty array ",
			exec: func() (any, error) {
				return CallPool[Float64_Array](pp, "identity", Float64_Array{})
			},
			want: Float64_Array{},
		},
		{
			name: "test float64 1D array ",
			exec: func() (any, error) {
				return CallPool[Float64_Array](pp, "identity", Float64_Array{1, 2, 3, 3, 4, 100})
			},
			want: Float64_Array{1, 2, 3, 3, 4, 100},
		},
		{
			name: "test float64 2D empty array ",
			exec: func() (any, error) {
				return CallPool[Float64_2DArray](pp, "identity", Float64_2DArray{})
			},
			want: Float64_2DArray{},
		},
		{
			name: "test float64 2D array ",
			exec: func() (any, error) {
				return CallPool[Float64_2DArray](pp, "identity", Float64_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}})
			},
			want: Float64_2DArray{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}},
		},
		{
			name: "test float64 3D empty array ",
			exec: func() (any, error) {
				return CallPool[Float64_3DArray](pp, "identity", Float64_3DArray{})
			},
			want: Float64_3DArray{},
		},
		{
			name: "test float64 3D array ",
			exec: func() (any, error) {
				return CallPool[Float64_3DArray](pp, "identity", Float64_3DArray{{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}}, {{1, 8, 5, 4, 1, 100}, {1, 8, 5, 4, 1, 99}}})
			},
			want: Float64_3DArray{{{1, 2, 3, 3, 4, 100}, {1, 8, 3, 3, 4, 100}}, {{1, 8, 5, 4, 1, 100}, {1, 8, 5, 4, 1, 99}}},
		},
		{
			name: "add two arrays",
			exec: func() (any, error) {
				return CallPool[Float64_Array](pp, "add_numpy_arrays", AddArraysInput{Float64_Array{5.5, 3.5}, Float64_Array{1, 2.1}})
			},
			want: Float64_Array{6.5, 5.6},
		},
		{
			name: "unknown function Call",
			exec: func() (any, error) {
				return CallPool[AddResult](pp, "add_scalar_output_blaba", AddInput{5, 6})
			},
			wantErr: true,
		},
		{
			name: "verify serialise float64 1d array",
			exec: func() (any, error) {
				return CallPool[ArrayWrapperFloat64](pp, "verify_1d_array", ArrayWrapperFloat64{Arr1D: Float64_Array{1.2, 3.2, 99.1, -14.1}})
			},
			wantErr: false,
			want:    ArrayWrapperFloat64{Arr1D: Float64_Array{1.2, 3.2, 99.1, -14.1}},
		},
		{
			name: "verify serialise float64 2d array",
			exec: func() (any, error) {
				return CallPool[ArrayWrapperFloat64](pp, "verify_2d_array", ArrayWrapperFloat64{Arr2D: Float64_2DArray{{1.2, 3.2}, {99.1, -14.1}}})
			},
			wantErr: false,
			want:    ArrayWrapperFloat64{Arr2D: Float64_2DArray{{1.2, 3.2}, {99.1, -14.1}}},
		},
		{
			name: "serialise/deserialise random array",
			exec: func() (any, error) {
				return CallPool[ArrayWrapperFloat64](pp, "identity", ArrayWrapperFloat64{Arr2D: testArr, Arr1D: testArr[0]})
			},
			wantErr: false,
			want:    ArrayWrapperFloat64{Arr2D: testArr, Arr1D: testArr[0]},
		},
		{
			name: "serialise/deserialise random float32 array",
			exec: func() (any, error) {
				return CallPool[ArrayWrapperFloat32](pp, "identity", ArrayWrapperFloat32{Arr2D: testArrFloat32, Arr1D: testArrFloat32[0]})
			},
			wantErr: false,
			want:    ArrayWrapperFloat32{Arr2D: testArrFloat32, Arr1D: testArrFloat32[0]},
		},
		{
			name: "serialise/deserialise basic",
			exec: func() (any, error) {
				return CallPool[ArrayWrapperFloat64](pp, "identity", ArrayWrapperFloat64{Arr2D: Float64_2DArray{{2.5, 1.34}, {1.1, 99.9}}})
			},
			wantErr: false,
			want:    ArrayWrapperFloat64{Arr2D: Float64_2DArray{{2.5, 1.34}, {1.1, 99.9}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.exec()
			if (err != nil) != tt.wantErr {
				t.Errorf("CallPool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CallPool() got = %v, want %v", got, tt.want)
			}
		})
	}
}
