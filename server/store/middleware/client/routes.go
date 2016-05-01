// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package client

import (
	"errors"
	"net/http"
	"net/url"
	"sync"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya/server/store"
)

// PrefixClient is the prefix to be used for client IDs.
const PrefixClient = "c-"

const pathClient = "/clients/:client"

var routesActivated sync.Once

func activateRoutes() {
	store.ActivateRoute(http.MethodPut, pathClient)
	store.ActivateRoute(http.MethodDelete, pathClient)
	store.ActivateRoute(http.MethodGet, pathClient)
}

func handleGetClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
	client, err := getClient(p)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	match, err := store.MustGetStore().HasString(PrefixClient + client)
	if err != nil {
		panic(err)
	}

	return http.StatusOK, store.ContainedResult{Contained: match}, nil
}

func handlePutClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	client, err := getClient(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = store.MustGetStore().PutString(PrefixClient + client)
	if err != nil {
		panic(err)
	}

	return http.StatusOK, nil
}

func handleDeleteClient(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	client, err := getClient(p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = store.MustGetStore().RemoveString(PrefixClient + client)
	if err != nil {
		if err == store.ErrResourceDoesNotExist {
			return http.StatusNotFound, err
		}
		panic(err)
	}

	return http.StatusOK, nil
}

func getClient(p httprouter.Params) (string, error) {
	clientString := p.ByName("client")
	if clientString == "" {
		return "", errors.New("misssing client")
	}

	clientString, err := url.QueryUnescape(clientString)
	if err != nil || len(clientString) != 6 {
		return "", errors.New("invalid client")
	}

	return clientString, nil
}
