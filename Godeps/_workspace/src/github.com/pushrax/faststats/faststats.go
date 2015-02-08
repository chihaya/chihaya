// Copyright 2015 The faststats Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package faststats

type Measure interface {
	AddSample(sample float64)
	Value() float64
}
