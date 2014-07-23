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

	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

const jsonContentType = "application/json; charset=UTF-8"

func (s *Server) check(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	if _, err := w.Write([]byte("STILL-ALIVE")); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	w.Header().Set("Content-Type", jsonContentType)

	e := json.NewEncoder(w)
	err := e.Encode(stats.DefaultStats)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	ann, err := NewAnnounce(s.config, r, p)
	writer := &Writer{w}

	if err == models.ErrMalformedRequest || err == models.ErrBadRequest {
		writer.WriteError(err)
		return http.StatusOK, nil
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	if err = s.tracker.HandleAnnounce(ann, writer); err != nil {
		return http.StatusInternalServerError, err
	}

	stats.RecordEvent(stats.Announce)

	return http.StatusOK, nil
}

func (s *Server) serveScrape(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	scrape, err := NewScrape(s.config, r, p)
	writer := &Writer{w}

	if err == models.ErrMalformedRequest {
		writer.WriteError(err)
		return http.StatusOK, nil
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	if err = s.tracker.HandleScrape(scrape, writer); err != nil {
		return http.StatusInternalServerError, err
	}

	stats.RecordEvent(stats.Scrape)

	return http.StatusOK, nil
}

func (s *Server) getTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	infohash, err := url.QueryUnescape(p.ByName("infohash"))
	if err != nil {
		return http.StatusNotFound, err
	}

	torrent, err := conn.FindTorrent(infohash)
	if err == models.ErrTorrentDNE {
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

func (s *Server) putTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var torrent models.Torrent
	err = json.Unmarshal(body, &torrent)
	if err != nil {
		return http.StatusBadRequest, err
	}

	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = conn.PutTorrent(&torrent)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) delTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	infohash, err := url.QueryUnescape(p.ByName("infohash"))
	if err != nil {
		return http.StatusNotFound, err
	}

	err = conn.DeleteTorrent(infohash)
	if err == models.ErrTorrentDNE {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	user, err := conn.FindUser(p.ByName("passkey"))
	if err == models.ErrUserDNE {
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

func (s *Server) putUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var user models.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return http.StatusBadRequest, err
	}

	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = conn.PutUser(&user)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) delUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = conn.DeleteUser(p.ByName("passkey"))
	if err == models.ErrUserDNE {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) putClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = conn.PutClient(p.ByName("clientID"))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) delClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = conn.DeleteClient(p.ByName("clientID"))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
