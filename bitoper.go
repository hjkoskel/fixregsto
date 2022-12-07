/*
Bit operations.
Used for arranging bits for better compression
*/
package fixregsto

import "fmt"

func boolsToBytes(t []bool) []byte {
	b := make([]byte, (len(t)+7)/8)
	for i, x := range t {
		if x {
			b[i/8] |= 0x80 >> uint(i%8)
		}
	}
	return b
}

func bytesToBools(b []byte) []bool {
	t := make([]bool, 8*len(b))
	for i, x := range b {
		for j := 0; j < 8; j++ {
			if (x<<uint(j))&0x80 == 0x80 {
				t[8*i+j] = true
			}
		}
	}
	return t
}

func sumIntArr(arr []int) int {
	result := 0
	for _, v := range arr {
		result += v
	}
	return result
}

func unsliceBitArr(sliced []byte, pattern []int) ([]byte, error) {
	if len(pattern) == 0 {
		return sliced, nil //NOP
	}

	inBools := bytesToBools(sliced)
	outBools := []bool{}

	structsize := sumIntArr(pattern)
	if len(inBools)%structsize != 0 {
		return nil, fmt.Errorf("got %v bits, must be multiple of pattern size %v", len(inBools), structsize)
	}

	structCount := len(inBools) / structsize
	structlist := make([][]bool, structCount)

	offset := 0
	for _, pat := range pattern {
		for structIndex := 0; structIndex < structCount; structIndex++ {
			structlist[structIndex] = append(structlist[structIndex], inBools[offset:offset+pat]...)
			offset += pat
		}
	}

	for _, arr := range structlist {
		outBools = append(outBools, arr...)
	}

	return boolsToBytes(outBools), nil
}

func sliceBitArr(arrIn []byte, pattern []int) ([]byte, error) {
	if len(pattern) == 0 {
		return arrIn, nil //NOP
	}
	inBools := bytesToBools(arrIn)
	outBools := []bool{}

	structsize := sumIntArr(pattern)
	if len(inBools)%structsize != 0 {
		return nil, fmt.Errorf("got %v bits, must be multiple of pattern size %v", len(inBools), structsize)
	}

	structCount := len(inBools) / structsize
	offset := 0 //inside struct
	for _, pat := range pattern {
		for structIndex := 0; structIndex < structCount; structIndex++ {
			startAddress := structsize*structIndex + offset
			outBools = append(outBools, inBools[startAddress:startAddress+pat]...)
		}
		offset += pat
	}
	if len(outBools) != len(inBools) {
		return nil, fmt.Errorf("internal error outbool len=%v inbool len=%v", len(outBools), len(inBools))
	}
	return boolsToBytes(outBools), nil
}
