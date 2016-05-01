package infohash

import (
	"encoding/base32"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"sync"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
)

// PrefixInfohash is the prefix to be used for infohashes.
const PrefixInfohash = "ih-"

const pathInfohash = "/infohashes/*infohash"

type infohashResult struct {
	InfoHash string `json:"infohash"`
}

type infohashContainedResult struct {
	infohashResult
	store.ContainedResult
}

var routesActivated sync.Once

func activateRoutes() {
	store.ActivateRoute(http.MethodPut, pathInfohash)
	store.ActivateRoute(http.MethodDelete, pathInfohash)
	store.ActivateRoute(http.MethodGet, pathInfohash)
}

func handleGetInfohash(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	infohash, err := getInfohash(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}
	result := infohashContainedResult{
		infohashResult: infohashResult{
			InfoHash: hex.EncodeToString([]byte(infohash[:])),
		},
	}

	match, err := store.MustGetStore().HasString(PrefixInfohash + string(infohash[:]))
	if err != nil {
		panic(err)
	}
	result.Contained = match

	return http.StatusOK, result, nil
}

func handlePutInfohash(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	infohash, err := getInfohash(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}
	result := infohashResult{
		InfoHash: hex.EncodeToString([]byte(infohash[:])),
	}

	err = store.MustGetStore().PutString(PrefixInfohash + string(infohash[:]))
	if err != nil {
		panic(err)
	}

	return http.StatusOK, result, nil
}

func handleDeleteInfohash(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	infohash, err := getInfohash(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}
	result := infohashResult{
		InfoHash: hex.EncodeToString([]byte(infohash[:])),
	}

	err = store.MustGetStore().RemoveString(PrefixInfohash + string(infohash[:]))
	if err != nil {
		if err == store.ErrResourceDoesNotExist {
			return http.StatusNotFound, result, err
		}
		panic(err)
	}

	return http.StatusOK, result, nil
}

func getInfohash(p httprouter.Params) (chihaya.InfoHash, error) {
	infohashString := p.ByName("infohash")
	if infohashString == "" {
		return chihaya.InfoHash{}, errors.New("missing infohash")
	}

	infohashString = infohashString[1:]

	var parsedInfohash string

	switch len(infohashString) {
	case 40:
		parsedBytes, err := hex.DecodeString(infohashString)
		if err != nil || len(parsedBytes) != 20 {
			return chihaya.InfoHash{}, errors.New("invalid infohash")
		}
		parsedInfohash = string(parsedBytes)
	case 32:
		parsedBytes, err := base32.StdEncoding.DecodeString(infohashString)
		if err != nil || len(parsedBytes) != 20 {
			return chihaya.InfoHash{}, errors.New("invalid infohash")
		}
		parsedInfohash = string(parsedBytes)
	default:
		parsedInfohash, err := url.QueryUnescape(infohashString)
		if err != nil || len(parsedInfohash) != 20 {
			return chihaya.InfoHash{}, errors.New("invalid infohash")
		}
	}

	return chihaya.InfoHashFromString(parsedInfohash), nil
}
