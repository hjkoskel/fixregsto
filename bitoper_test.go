package fixregsto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlicing(t *testing.T) {
	testpattern := []int{8, 8}
	inputdata := []byte{1, 2, 3, 4}
	result, err := sliceBitArr(inputdata, testpattern)
	assert.Equal(t, nil, err)
	assert.Equal(t, []byte{0x1, 0x3, 0x2, 0x4}, result)
	inputBack, errUn := unsliceBitArr(result, testpattern)
	assert.Equal(t, nil, errUn)
	assert.Equal(t, inputdata, inputBack)
}

func byteArrToBinaryString(arr []byte) string {
	s := ""
	for _, b := range arr {
		s += fmt.Sprintf("%08b", b)
	}
	return s
}

func TestSlicing2(t *testing.T) {
	testpattern := []int{8, 8, 7, 1}
	inputdata := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	result, err := sliceBitArr(inputdata, testpattern)
	assert.Equal(t, nil, err)
	/*
		fmt.Printf("input  %s\n", byteArrToBinaryString(inputdata))
		fmt.Printf("result %s\n", byteArrToBinaryString(result))

			-- in binary format 8,8,7,1 pieces ---
			00000001
			00000010
			0000001
			1
			00000100
			00000101
			0000011
			0
			00000111
			00001000
			0000100
			1
			00001010
			00001011
			0000110
			0
			-- after slicing
			00000001
			00000100
			00000111
			00001010

			00000010
			00000101
			00001000
			00001011

			0000001
			0000011
			0000100
			0000110

			1
			0
			1
			0
	*/

	assert.Equal(t, "000000010000010000000111000010100000001000000101000010000000101100000010000011000010000001101010", byteArrToBinaryString(result))
	inputBack, errUn := unsliceBitArr(result, testpattern)

	assert.Equal(t, nil, errUn)
	assert.Equal(t, inputdata, inputBack)
}
