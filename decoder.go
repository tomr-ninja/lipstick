package lipstick

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"io"
)

// Decoder implements Decoder
type Decoder struct {
	order     int
	frameSize int
	history   []float64
	huffman   interface {
		io.ReadCloser
		flate.Resetter
	}
	bufs struct {
		lpc        []float64
		qlpc       []int16
		res        []float64
		qres       []int8
		qresBinned []uint8
	}
}

func NewDecoder(order, frameSize int) *Decoder {
	if frameSize <= order {
		panic("frameSize must be > order")
	}

	huffman := flate.NewReader(nil).(interface {
		io.ReadCloser
		flate.Resetter
	})

	return &Decoder{
		order:     order,
		frameSize: frameSize,
		history:   make([]float64, order),
		huffman:   huffman,
		bufs: struct {
			lpc        []float64
			qlpc       []int16
			res        []float64
			qres       []int8
			qresBinned []uint8
		}{
			lpc:        make([]float64, order),
			qlpc:       make([]int16, order),
			res:        make([]float64, frameSize),
			qres:       make([]int8, frameSize),
			qresBinned: make([]uint8, frameSize),
		},
	}
}

func (d *Decoder) Decode(from []byte, to []float32) error {
	// unpack, reconstruct residual, synthesize x[n] = e[n] - sum lpc[k]*x[n-k]
	if len(from) < 4 {
		return errors.New("frame too short")
	}
	buf := bytes.NewReader(from)

	// read header
	magic := make([]byte, 4)
	if _, err := buf.Read(magic); err != nil {
		return err
	}
	if ms := string(magic); ms != magicHeader {
		if ms == silentHeader {
			for i := 0; i < d.frameSize; i++ {
				to[i] = 0
			}
			return nil
		}
		return errors.New("invalid magic header")
	}
	orderByte, err := buf.ReadByte()
	if err != nil {
		return err
	}
	order := int(orderByte)

	lpc := d.bufs.lpc[:order]
	qlpc := d.bufs.qlpc[:order]
	if order <= 3 {
		if err = bufReadQuantized(buf, lpc, qlpc, dequantize16); err != nil {
			return err
		}
	} else {
		if err = bufReadQuantized(buf, lpc[:3], qlpc[:3], dequantize16); err != nil {
			return err
		}
		if err = bufReadQuantized(buf, lpc[3:], qlpc[3:], dequantize16); err != nil {
			return err
		}
	}

	// read residual scale
	var scaleRes float32
	if err = binary.Read(buf, binary.LittleEndian, &scaleRes); err != nil {
		return err
	}
	if scaleRes == 0 {
		scaleRes = 1e-9
	}

	// read residual compressed bytes
	var residualSize uint16
	if err = binary.Read(buf, binary.LittleEndian, &residualSize); err != nil {
		return err
	}

	_ = d.huffman.Reset(io.LimitReader(buf, int64(residualSize)), nil)
	qresBinned := d.bufs.qresBinned
	if err = binary.Read(d.huffman, binary.LittleEndian, &qresBinned); err != nil {
		return err
	}

	// Reconstruct residual values back to float64
	qres := d.bufs.qres
	unbin(qresBinned, qres)
	residual := d.bufs.res
	dequantize8(scaleRes, qres, residual)

	// Synthesize x[n] = e[n] - sum_{k=1..p} lpc[k]*x[n-k]
	x := make([]float64, d.frameSize)
	for n := 0; n < d.frameSize; n++ {
		var pred float64
		for k := 1; k <= d.order; k++ {
			idx := n - k
			var sample float64
			if idx >= 0 {
				sample = x[idx]
			} else {
				hIdx := d.order + idx
				if hIdx >= 0 {
					sample = d.history[hIdx]
				}
			}
			pred += lpc[k-1] * sample
		}
		// inverse of encoder residual definition: x[n] = residual[n] - pred
		x[n] = residual[n] - pred
	}
	copy(d.history, x[d.frameSize-d.order:])

	for i := 0; i < d.frameSize; i++ {
		to[i] = float32(x[i])
	}

	return nil
}
