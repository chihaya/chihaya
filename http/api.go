// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"
)

func (t *Tracker) getTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	torrent, err := conn.FindTorrent(p.ByName("infohash"))
	if err == tracker.ErrTorrentDNE {
		return http.StatusNotFound
	} else if err != nil {
		return http.StatusInternalServerError
	}

	e := json.NewEncoder(w)
	err = e.Encode(torrent)
	if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (t *Tracker) putTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError
	}

	var torrent models.Torrent
	err = json.Unmarshal(body, &torrent)
	if err != nil {
		return http.StatusBadRequest
	}

	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	err = conn.PutTorrent(&torrent)
	if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (t *Tracker) delTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	err = conn.DeleteTorrent(p.ByName("infohash"))
	if err == tracker.ErrTorrentDNE {
		return http.StatusNotFound
	} else if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (t *Tracker) getUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	user, err := conn.FindUser(p.ByName("passkey"))
	if err == tracker.ErrUserDNE {
		return http.StatusNotFound
	} else if err != nil {
		return http.StatusInternalServerError
	}

	e := json.NewEncoder(w)
	err = e.Encode(user)
	if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (t *Tracker) putUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError
	}

	var user models.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return http.StatusBadRequest
	}

	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	err = conn.PutUser(&user)
	if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (t *Tracker) delUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	err = conn.DeleteUser(p.ByName("passkey"))
	if err == tracker.ErrUserDNE {
		return http.StatusNotFound
	} else if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (t *Tracker) putClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	err = conn.PutClient(p.ByName("clientID"))
	if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (t *Tracker) delClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) int {
	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError
	}

	err = conn.DeleteClient(p.ByName("clientID"))
	if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}
