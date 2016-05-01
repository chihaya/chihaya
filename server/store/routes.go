// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/chihaya/chihaya"
	"github.com/julienschmidt/httprouter"
)

// ResponseFunc is the type of function that handles an API request and returns
// an HTTP status code, an optional response to be embedded and an error.
type ResponseFunc func(http.ResponseWriter, *http.Request, httprouter.Params) (status int, result interface{}, err error)

// NoResultResponseFunc is the type of function that handles an API request and
// returns an HTTP status code and an error.
type NoResultResponseFunc func(http.ResponseWriter, *http.Request, httprouter.Params) (status int, err error)

// ErrInternalServerError is the error used for recovered API calls.
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

type countResult struct {
	Count int `json:"count"`
}

type peersResult struct {
	Peers4 []peer `json:"peers4"`
	Peers6 []peer `json:"peers6"`
}

type peer struct {
	ID   string `json:"id"`
	IP   string `json:"ip"`
	Port uint16 `json:"port"`
}

type dualStackedPeer struct {
	Peer4 peer `json:"peer4"`
	Peer6 peer `json:"peer6"`
}

func (s *Store) makeHandler(inner ResponseFunc) httprouter.Handle {
	if s.cfg.APIKey != "" {
		inner = authorizationHandler(inner, s.cfg.APIKey)
	}

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

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		w.WriteHeader(status)

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Println("API: unable to send response:", err)
		}
	}
}

func authorizationHandler(inner ResponseFunc, apiKey string) ResponseFunc {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
		token := getAPIKey(r)
		if token != apiKey {
			return http.StatusForbidden, nil, errors.New("invalid API key")
		}

		return inner(w, r, p)
	}
}

func getAPIKey(r *http.Request) string {
	token := r.Header.Get("X-API-Key")

	if token == "" {
		token = r.URL.Query().Get("apikey")
	}

	return token
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

func (s *Store) handleGetSeeders(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	return s.handleGetPeers(w, r, p, true)
}

func (s *Store) handleGetLeechers(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	return s.handleGetPeers(w, r, p, false)
}

func (s *Store) handleGetPeers(w http.ResponseWriter, r *http.Request, p httprouter.Params, seeders bool) (int, interface{}, error) {
	ih, err := getInfohash(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	var peers4, peers6 []chihaya.Peer
	if seeders {
		peers4, peers6, err = s.GetSeeders(ih)
	} else {
		peers4, peers6, err = s.GetLeechers(ih)
	}
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, nil, err
		}
		panic(err)
	}

	return http.StatusOK, toPeersResult(peers4, peers6), nil
}

func (s *Store) handlePutSeeder(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	return s.handlePutPeer(w, r, p, true)
}

func (s *Store) handlePutLeecher(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	return s.handlePutPeer(w, r, p, false)
}

func (s *Store) handlePutPeer(w http.ResponseWriter, r *http.Request, params httprouter.Params, seeder bool) (int, error) {
	ih, err := getInfohash(params)
	if err != nil {
		return http.StatusBadRequest, err
	}

	rawPeer := peer{}
	err = json.NewDecoder(r.Body).Decode(&rawPeer)
	if err != nil {
		return http.StatusBadRequest, errors.New("invalid peer")
	}

	p, err := decodePeer(rawPeer)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if seeder {
		err = s.PutSeeder(ih, p)
	} else {
		err = s.PutLeecher(ih, p)
	}
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, err
		}
		panic(err)
	}

	return http.StatusOK, nil
}

func (s *Store) handleDeleteSeeder(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	return s.handleDeletePeer(w, r, p, true)
}

func (s *Store) handleDeleteLeecher(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	return s.handleDeletePeer(w, r, p, false)
}

func (s *Store) handleDeletePeer(w http.ResponseWriter, r *http.Request, params httprouter.Params, seeder bool) (int, error) {
	ih, err := getInfohash(params)
	if err != nil {
		return http.StatusBadRequest, err
	}

	rawPeer := peer{}
	err = json.NewDecoder(r.Body).Decode(&rawPeer)
	if err != nil {
		return http.StatusBadRequest, errors.New("invalid peer")
	}

	p, err := decodePeer(rawPeer)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if seeder {
		err = s.DeleteSeeder(ih, p)
	} else {
		err = s.DeleteLeecher(ih, p)
	}
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, err
		}
		panic(err)
	}

	return http.StatusOK, nil
}

func (s *Store) handleNumSeeders(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	return s.handleNumPeers(w, r, p, true)
}

func (s *Store) handleNumLeechers(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	return s.handleNumPeers(w, r, p, false)
}

func (s *Store) handleNumPeers(w http.ResponseWriter, r *http.Request, p httprouter.Params, seeders bool) (int, interface{}, error) {
	ih, err := getInfohash(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	var count int
	if seeders {
		count = s.NumSeeders(ih)
	} else {
		count = s.NumLeechers(ih)
	}

	return http.StatusOK, countResult{Count: count}, nil
}

func (s *Store) handleGraduateLeecher(w http.ResponseWriter, r *http.Request, params httprouter.Params) (int, error) {
	ih, err := getInfohash(params)
	if err != nil {
		return http.StatusBadRequest, err
	}

	rawPeer := peer{}
	err = json.NewDecoder(r.Body).Decode(&rawPeer)
	if err != nil {
		return http.StatusBadRequest, errors.New("invalid peer")
	}

	p, err := decodePeer(rawPeer)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.GraduateLeecher(ih, p)
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, err
		}
		panic(err)
	}

	return http.StatusOK, nil
}

func (s *Store) handleAnnounce(w http.ResponseWriter, r *http.Request, params httprouter.Params) (int, interface{}, error) {
	ih, err := getInfohash(params)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	isSeeder := params.ByName("seeder")
	seeder, err := strconv.ParseBool(isSeeder)
	if err != nil {
		return http.StatusBadRequest, nil, errors.New("invalid value for seeder")
	}

	rawPeer := dualStackedPeer{}
	err = json.NewDecoder(r.Body).Decode(&rawPeer)
	if err != nil {
		return http.StatusBadRequest, nil, errors.New("invalid peer")
	}

	var p4, p6 chihaya.Peer

	if rawPeer.Peer4.IP != "" {
		p4, err = decodePeer(rawPeer.Peer4)
		if err != nil {
			return http.StatusBadRequest, nil, err
		}
	}

	if rawPeer.Peer6.IP != "" {
		p6, err = decodePeer(rawPeer.Peer6)
		if err != nil {
			return http.StatusBadRequest, nil, err
		}
	}

	peers4, peers6, err := s.AnnouncePeers(ih, seeder, 50, p4, p6)
	if err != nil {
		if err == ErrResourceDoesNotExist {
			return http.StatusNotFound, nil, err
		}
		panic(err)
	}

	return http.StatusOK, toPeersResult(peers4, peers6), nil
}

func toPeersResult(peers4, peers6 []chihaya.Peer) peersResult {
	toReturn := peersResult{
		Peers4: make([]peer, len(peers4)),
		Peers6: make([]peer, len(peers6)),
	}

	for i, p := range peers4 {
		toReturn.Peers4[i] = encodePeer(p)
	}

	for i, p := range peers6 {
		toReturn.Peers6[i] = encodePeer(p)
	}

	return toReturn
}

func encodePeer(p chihaya.Peer) peer {
	return peer{
		ID:   hex.EncodeToString([]byte(p.ID[:])),
		IP:   p.IP.String(),
		Port: p.Port,
	}
}

func decodePeer(p peer) (chihaya.Peer, error) {
	var parsedPeerID []byte
	var err error

	switch len(p.ID) {
	case 40:
		parsedPeerID, err = hex.DecodeString(p.ID)
	case 32:
		parsedPeerID, err = base32.StdEncoding.DecodeString(p.ID)
	case 20:
		parsedPeerID = []byte(p.ID)
	default:
		return chihaya.Peer{}, errors.New("invalid peer ID")
	}
	if err != nil || len(parsedPeerID) != 20 {
		return chihaya.Peer{}, errors.New("invalid peer ID")
	}

	ip := net.ParseIP(p.IP)
	if ip == nil {
		return chihaya.Peer{}, errors.New("invalid IP")
	}

	return chihaya.Peer{
		ID:   chihaya.PeerIDFromBytes(parsedPeerID),
		IP:   ip,
		Port: p.Port,
	}, nil
}

func getInfohash(p httprouter.Params) (chihaya.InfoHash, error) {
	infohashString := p.ByName("infohash")
	if infohashString == "" {
		return chihaya.InfoHash{}, errors.New("missing infohash")
	}

	infohashString = infohashString[1:]

	var (
		parsedBytes    []byte
		parsedInfohash string
		err            error
	)

	switch len(infohashString) {
	case 40:
		parsedBytes, err = hex.DecodeString(infohashString)
		if err != nil || len(parsedBytes) != 20 {
			break
		}
		parsedInfohash = string(parsedBytes)
	case 32:
		parsedBytes, err = base32.StdEncoding.DecodeString(infohashString)
		if err != nil || len(parsedBytes) != 20 {
			break
		}
		parsedInfohash = string(parsedBytes)
	default:
	}

	if err != nil || len(parsedInfohash) != 20 {
		// always try URLEncoding, no matter the length.
		parsedInfohash, err = url.QueryUnescape(infohashString)
		if err != nil || len(parsedInfohash) != 20 {
			return chihaya.InfoHash{}, errors.New("invalid infohash")
		}
	}

	return chihaya.InfoHashFromString(parsedInfohash), nil
}
