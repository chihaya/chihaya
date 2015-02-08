// Copyright 2015 The faststats Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package faststats

import "encoding/json"

func (p *Percentile) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Value())
}
