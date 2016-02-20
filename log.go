package cml

import (
	"errors"
	"math"
)

/*
Sketch is a Count-Min-Log Sketch 16-bit registers
*/
type Sketch struct {
	w   uint
	d   uint
	exp float64

	store [][]uint16
}

/*
NewSketch returns a new Count-Min-Log Sketch with 16-bit registers
*/
func NewSketch(w uint, d uint, exp float64) (*Sketch, error) {
	store := make([][]uint16, d, d)
	for i := uint(0); i < d; i++ {
		store[i] = make([]uint16, w, w)
	}
	return &Sketch{
		w:     w,
		d:     d,
		exp:   exp,
		store: store,
	}, nil
}

/*
NewSketchForEpsilonDelta ...
*/
func NewSketchForEpsilonDelta(epsilon, delta float64) (*Sketch, error) {
	var (
		width = uint(math.Ceil(math.E / epsilon))
		depth = uint(math.Ceil(math.Log(1 / delta)))
	)
	return NewSketch(width, depth, 1.00026)
}

/*
NewDefaultSketch returns a new Count-Min-Log Sketch with 16-bit registers and default settings
*/
func NewDefaultSketch() (*Sketch, error) {
	return NewSketch(1000000, 7, 1.00026)
}

/*
NewForCapacity16 returns a new Count-Min-Log Sketch with 16-bit registers optimized for a given max capacity and expected error rate
*/
func NewForCapacity16(capacity uint64, e float64) (*Sketch, error) {
	if !(e >= 0.001 && e < 1.0) {
		return nil, errors.New("e needs to be >= 0.001 and < 1.0")
	}
	if capacity < 1000000 {
		capacity = 1000000
	}
	w := float64(capacity) / 256
	return NewSketch(uint(w), 32, 1.00026)
}

func (cml *Sketch) increaseDecision(c uint16) bool {
	return randFloat() < 1/math.Pow(cml.exp, float64(c))
}

/*
Update increases the count of `s` by one, return true if added and the current count of `s`
*/
func (cml *Sketch) Update(e []byte) bool {
	sk := make([]*uint16, cml.d, cml.d)
	c := uint16(math.MaxUint16)
	for i := range sk {
		if sk[i] = &cml.store[i][hash(e, uint(i), cml.w)]; *sk[i] < c {
			c = *sk[i]
		}
	}

	if cml.increaseDecision(c) {
		for _, k := range sk {
			if *k == c {
				*k = c + 1
			}
		}
	}
	return true
}

/*
BulkUpdate increases the count of `s` by one, return true if added and the current count of `s`
*/
func (cml *Sketch) BulkUpdate(e []byte, freq uint) bool {
	sk := make([]*uint16, cml.d, cml.d)
	c := uint16(math.MaxUint16)
	for i := range sk {
		if sk[i] = &cml.store[i][hash(e, uint(i), cml.w)]; *sk[i] < c {
			c = *sk[i]
		}
	}

	for i := uint(0); i < freq; i++ {
		update := false
		if cml.increaseDecision(c) {
			for _, k := range sk {
				if *k == c {
					*k = c + 1
					update = true
				}
			}
		}
		if update {
			c++
		}
	}
	return true
}

func (cml *Sketch) pointValue(c uint16) float64 {
	if c == 0 {
		return 0
	}
	return math.Pow(cml.exp, float64(c-1))
}

func (cml *Sketch) value(c uint16) float64 {
	if c <= 1 {
		return cml.pointValue(c)
	}
	v := cml.pointValue(c + 1)
	return (1 - v) / (1 - cml.exp)
}

/*
Query returns the count of `e`
*/
func (cml *Sketch) Query(e []byte) float64 {
	c := uint16(math.MaxUint16)
	for i := range cml.store {
		if sk := cml.store[i][hash(e, uint(i), cml.w)]; sk < c {
			c = sk
		}
	}
	return cml.value(c)
}
