// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/julienschmidt/httprouter"
)

// ResponseFunc is the type of function that handles an API request and returns
// an HTTP status code, an optional response to be embedded and an error.
type ResponseFunc func(http.ResponseWriter, *http.Request, httprouter.Params) (status int, result interface{}, err error)

// NoResultResponseFunc is the type of function that handles an API request and
// returns an HTTP status code and an error.
type NoResultResponseFunc func(http.ResponseWriter, *http.Request, httprouter.Params) (status int, err error)

var ErrInternalServerError = errors.New("internal server error")

type response struct {
	Ok     bool        `json:"ok"`
	Error  string      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

// ContainedResult is the result returned by endpoints that check if a given
// entity is contained in the store.
type ContainedResult struct {
	Contained bool `json:"contained"`
}

func makeHandler(inner ResponseFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		resp := response{}
		handler := logHandler(recoverHandler(inner))

		status, result, err := handler(w, r, p)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Ok = true
		}
		if result != nil {
			resp.Result = result
		}

		w.WriteHeader(status)

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			panic(err)
		}
	}
}

func logHandler(inner ResponseFunc) ResponseFunc {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
		before := time.Now()

		status, result, err := inner(w, r, p)
		delta := time.Since(before)

		log.Printf("%d %s %s %s %s", status, delta.String(), r.RemoteAddr, r.Method, r.URL.EscapedPath())

		return status, result, err
	}
}

func recoverHandler(inner ResponseFunc) ResponseFunc {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) (status int, result interface{}, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Println("API: recovered:", rec)
				status = http.StatusInternalServerError
				result = nil
				err = ErrInternalServerError
			}
		}()

		status, result, err = inner(w, r, p)
		return
	}
}

func noResultHandler(inner NoResultResponseFunc) ResponseFunc {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
		status, err := inner(w, r, p)

		return status, nil, err
	}
}

func (s *Store) handleGetIP(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	ip, err := getIP(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	match, err := s.HasIP(ip)
	if err != nil {
		panic(err)
	}

	return http.StatusOK, ContainedResult{Contained: match}, nil
}

func (s *Store) handlePutIP(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	ip, err := getIP(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.AddIP(ip)
	if err != nil {
		panic(err)
	}

	return http.StatusOK, nil
}

func (s *Store) handleDeleteIP(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	ip, err := getIP(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.RemoveIP(ip)
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, err
		}
		panic(err)
	}

	return http.StatusOK, nil
}

func getIP(p httprouter.Params) (net.IP, error) {
	ipString := p.ByName("ip")
	if ipString == "" {
		return nil, errors.New("missing IP")
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return nil, errors.New("invalid IP")
	}

	return ip, nil
}

func (s *Store) handlePutNetwork(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	network, err := getNetwork(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.AddNetwork(network)
	if err != nil {
		panic(err)
	}

	return http.StatusOK, nil
}

func (s *Store) handleDeleteNetwork(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	network, err := getNetwork(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.RemoveNetwork(network)
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, err
		}
		panic(err)
	}

	return http.StatusOK, nil
}

func getNetwork(p httprouter.Params) (string, error) {
	networkString := p.ByName("network")
	if len(networkString) < 2 {
		return "", errors.New("missing network")
	}

	// Remove preceding slash.
	networkString = networkString[1:]

	_, _, err := net.ParseCIDR(networkString)
	if err != nil {
		return "", errors.New("invalid network")
	}

	return networkString, nil
}

func (s *Store) handleGetString(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	str, err := getString(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	match, err := s.HasString(str)
	if err != nil {
		panic(err)
	}

	return http.StatusOK, ContainedResult{Contained: match}, nil
}

func (s *Store) handlePutString(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	str, err := getString(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.PutString(str)
	if err != nil {
		panic(err)
	}

	return http.StatusOK, nil
}

func (s *Store) handleDeleteString(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	str, err := getString(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.RemoveString(str)
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, err
		}
		panic(err)
	}

	return http.StatusOK, nil
}

func getString(p httprouter.Params) (string, error) {
	str := p.ByName("string")
	if str == "" {
		return "", errors.New("missing string")
	}

	str = str[1:]
	parsed, err := url.QueryUnescape(str)
	if err != nil {
		return "", errors.New("invalid string")
	}

	return parsed, nil
}
