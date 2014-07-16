// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"
)

const jsonContentType = "application/json; charset=UTF-8"

func (t *Tracker) check(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	_, err := w.Write([]byte("An easter egg goes here."))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) getTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	infohash, err := url.QueryUnescape(p.ByName("infohash"))
	if err != nil {
		return http.StatusNotFound, err
	}

	torrent, err := conn.FindTorrent(infohash)
	if err == tracker.ErrTorrentDNE {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", jsonContentType)
	e := json.NewEncoder(w)
	err = e.Encode(torrent)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) putTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var torrent models.Torrent
	err = json.Unmarshal(body, &torrent)
	if err != nil {
		return http.StatusBadRequest, err
	}

	conn, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = conn.PutTorrent(&torrent)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) delTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	infohash, err := url.QueryUnescape(p.ByName("infohash"))
	if err != nil {
		return http.StatusNotFound, err
	}

	err = conn.DeleteTorrent(infohash)
	if err == tracker.ErrTorrentDNE {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) getUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	base, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	conn, private := base.(tracker.PrivateConn)
	if !private {
		return http.StatusNotFound, nil
	}

	user, err := conn.FindUser(p.ByName("passkey"))
	if err == tracker.ErrUserDNE {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", jsonContentType)
	e := json.NewEncoder(w)
	err = e.Encode(user)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) putUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var user models.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return http.StatusBadRequest, err
	}

	base, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	conn, private := base.(tracker.PrivateConn)
	if !private {
		return http.StatusNotFound, nil
	}

	err = conn.PutUser(&user)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) delUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	base, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	conn, private := base.(tracker.PrivateConn)
	if !private {
		return http.StatusNotFound, nil
	}

	err = conn.DeleteUser(p.ByName("passkey"))
	if err == tracker.ErrUserDNE {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) putClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	base, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	conn, private := base.(tracker.PrivateConn)
	if !private {
		return http.StatusNotFound, nil
	}

	err = conn.PutClient(p.ByName("clientID"))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *Tracker) delClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	base, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	conn, private := base.(tracker.PrivateConn)
	if !private {
		return http.StatusNotFound, nil
	}

	err = conn.DeleteClient(p.ByName("clientID"))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
