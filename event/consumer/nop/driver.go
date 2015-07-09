// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package nop implements a Consumer as a no-op.
package nop

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/event/consumer"
	"github.com/chihaya/chihaya/tracker/models"
)

type driver struct{}

// Nop is a Consumer for a BitTorrent tracker that does nothing.
type Nop struct{}

func (d *driver) New(cfg *config.DriverConfig) (consumer.Consumer, error) {
	return &Nop{}, nil
}

// Close returns nil.
func (n *Nop) Close() error {
	return nil
}

// RecordAnnounce returns nil.
func (n *Nop) RecordAnnounce(delta *models.AnnounceDelta) error {
	return nil
}

// Init registers the nop driver.
func init() {
	consumer.Register("nop", &driver{})
}
