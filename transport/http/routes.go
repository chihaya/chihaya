// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"

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
	// Attempt to ping the backend if private tracker is enabled.
	if s.config.PrivateEnabled {
		if err := s.tracker.DeltaStore.Ping(); err != nil {
			return handleError(err)
		}
	}

	_, err := w.Write([]byte("STILL-ALIVE"))
	return handleError(err)
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	w.Header().Set("Content-Type", jsonContentType)

	var err error
	var val interface{}
	query := r.URL.Query()

	stats.DefaultStats.GoRoutines = runtime.NumGoroutine()

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
	} else if models.IsPublicError(err) {
		w.WriteError(err)
		stats.RecordEvent(stats.ClientError)
		return http.StatusOK, nil
	}

	return http.StatusInternalServerError, err
}

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	writer := &Writer{w}
	ann, err := s.newAnnounce(r, p)
	if err != nil {
		return handleTorrentError(err, writer)
	}

	return handleTorrentError(s.tracker.HandleAnnounce(ann, writer), writer)
}

func (s *Server) serveScrape(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	writer := &Writer{w}
	scrape, err := s.newScrape(r, p)
	if err != nil {
		return handleTorrentError(err, writer)
	}

	return handleTorrentError(s.tracker.HandleScrape(scrape, writer), writer)
}

func (s *Server) getTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	infohash, err := url.QueryUnescape(p.ByName("infohash"))
	if err != nil {
		return http.StatusNotFound, err
	}

	torrent, err := s.tracker.Store.FindTorrent(infohash)
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

	s.tracker.Store.PutTorrent(&torrent)
	return http.StatusOK, nil
}

func (s *Server) delTorrent(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	infohash, err := url.QueryUnescape(p.ByName("infohash"))
	if err != nil {
		return http.StatusNotFound, err
	}

	s.tracker.Store.DeleteTorrent(infohash)
	return http.StatusOK, nil
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	user, err := s.tracker.Store.FindUser(p.ByName("passkey"))
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

	s.tracker.Store.PutUser(&user)
	return http.StatusOK, nil
}

func (s *Server) delUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	s.tracker.Store.DeleteUser(p.ByName("passkey"))
	return http.StatusOK, nil
}

func (s *Server) getClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	if err := s.tracker.Store.FindClient(p.ByName("clientID")); err != nil {
		return http.StatusNotFound, err
	}
	return http.StatusOK, nil
}

func (s *Server) putClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	s.tracker.Store.PutClient(p.ByName("clientID"))
	return http.StatusOK, nil
}

func (s *Server) delClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	s.tracker.Store.DeleteClient(p.ByName("clientID"))
	return http.StatusOK, nil
}
