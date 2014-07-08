// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (t *Tracker) getTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {}
func (t *Tracker) putTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {}
func (t *Tracker) delTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {}

func (t *Tracker) getUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {}
func (t *Tracker) putUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {}
func (t *Tracker) delUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {}
