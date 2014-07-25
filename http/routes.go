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

func handleError(err error) (int, error) {
	if err == nil {
		return http.StatusOK, nil
	} else if _, ok := err.(models.NotFoundError); ok {
		stats.RecordEvent(stats.ClientError)
		return http.StatusNotFound, nil
	} else if _, ok := err.(models.ClientError); ok {
		stats.RecordEvent(stats.ClientError)
		return http.StatusBadRequest, nil
	}
	return http.StatusInternalServerError, err
}

func (s *Server) check(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	_, err := w.Write([]byte("STILL-ALIVE"))
	return handleError(err)
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	w.Header().Set("Content-Type", jsonContentType)

	var err error
	var val interface{}
	query := r.URL.Query()

	if _, flatten := query["flatten"]; flatten {
		val = stats.DefaultStats.Flattened()
	} else {
		val = stats.DefaultStats
	}

	if _, pretty := query["pretty"]; pretty {
		var buf []byte
		buf, err = json.MarshalIndent(val, "", "  ")

		if err == nil {
			_, err = w.Write(buf)
		}
	} else {
		err = json.NewEncoder(w).Encode(val)
	}

	return handleError(err)
}

func handleTorrentError(err error, w *Writer) (int, error) {
	if err == nil {
		return http.StatusOK, nil
	} else if _, ok := err.(models.ClientError); ok {
		w.WriteError(err)
		stats.RecordEvent(stats.ClientError)
		return http.StatusOK, nil
	}
	return http.StatusInternalServerError, err
}

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	stats.RecordEvent(stats.Announce)

	writer := &Writer{w}
	ann, err := NewAnnounce(s.config, r, p)
	if err != nil {
		return handleTorrentError(err, writer)
	}

	return handleTorrentError(s.tracker.HandleAnnounce(ann, writer), writer)
}

func (s *Server) serveScrape(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	stats.RecordEvent(stats.Scrape)

	writer := &Writer{w}
	scrape, err := NewScrape(s.config, r, p)
	if err != nil {
		return handleTorrentError(err, writer)
	}

	return handleTorrentError(s.tracker.HandleScrape(scrape, writer), writer)
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
	if err != nil {
		return handleError(err)
	}

	w.Header().Set("Content-Type", jsonContentType)
	e := json.NewEncoder(w)
	return handleError(e.Encode(torrent))
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

	return handleError(conn.PutTorrent(&torrent))
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

	return handleError(conn.DeleteTorrent(infohash))
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
	return handleError(e.Encode(user))
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

	return handleError(conn.PutUser(&user))
}

func (s *Server) delUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return handleError(conn.DeleteUser(p.ByName("passkey")))
}

func (s *Server) putClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return handleError(conn.PutClient(p.ByName("clientID")))
}

func (s *Server) delClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	conn, err := s.tracker.Pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return handleError(conn.DeleteClient(p.ByName("clientID")))
}
