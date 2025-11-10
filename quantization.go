package lipstick

import (
	"encoding/binary"
	"io"
	"math"
)

func quantize8(vs []float64, quantized []int8) (scale float32) {
	scale64 := 1e-9
	for _, v := range vs {
		av := math.Abs(v)
		if av > scale64 {
			scale64 = av
		}
	}
	scale = float32(scale64)

	for i := 0; i < len(quantized); i++ {
		q := int(math.Round(vs[i] / scale64 * 127.0))
		if q > 127 {
			q = 127
		}
		if q < -128 {
			q = -128
		}
		quantized[i] = int8(q)
	}

	return scale
}

func dequantize8(scale float32, qvs []int8, to []float64) {
	if len(to) != len(qvs) {
		panic("to and qvs must have the same length")
	}

	scale64 := float64(scale)
	for i := 0; i < len(to); i++ {
		to[i] = float64(qvs[i]) * scale64 / 127.0
	}
}

func quantize16(vs []float64, quantized []int16) (scale float32) {
	scale64 := 1e-9
	for _, v := range vs {
		av := math.Abs(v)
		if av > scale64 {
			scale64 = av
		}
	}
	scale = float32(scale64)

	for i := 0; i < len(quantized); i++ {
		q := int(math.Round(vs[i] / scale64 * 32767.0))
		if q > 32767 {
			q = 32767
		}
		if q < -32768 {
			q = -32768
		}
		quantized[i] = int16(q)
	}

	return scale
}

func dequantize16(scale float32, qvs []int16, to []float64) {
	if len(to) != len(qvs) {
		panic("to and qvs must have the same length")
	}

	scale64 := float64(scale)
	for i := 0; i < len(to); i++ {
		to[i] = float64(qvs[i]) * scale64 / 32767.0
	}
}

// copied from https://pkg.go.dev/golang.org/x/exp/constraints#Integer
type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func bufWriteQuantized[T integer](
	buf io.Writer,
	vs []float64,
	qvs []T,
	qf func([]float64, []T) float32,
) error {
	if len(vs) != len(qvs) {
		panic("vs and qvs must have the same length")
	}

	scale := qf(vs, qvs)

	if err := binary.Write(buf, binary.LittleEndian, scale); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, qvs); err != nil {
		return err
	}

	return nil
}

func bufReadQuantized[T integer](
	buf io.Reader,
	vs []float64,
	qvs []T,
	rqf func(float32, []T, []float64)) error {
	if len(vs) != len(qvs) {
		panic("vs and qvs must have the same length")
	}

	var scale float32
	if err := binary.Read(buf, binary.LittleEndian, &scale); err != nil {
		return err
	}
	if scale == 0 {
		scale = 1e-9
	}

	if err := binary.Read(buf, binary.LittleEndian, &qvs); err != nil {
		return err
	}

	rqf(scale, qvs, vs)

	return nil
}
