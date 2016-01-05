// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	oidchttp "github.com/coreos/go-oidc/http"
	"github.com/coreos/go-oidc/jose"
	"github.com/golang/glog"
)

const jwkTTLFallback = 5 * time.Minute

func (tkr *Tracker) updateJWKSetForever() {
	defer tkr.shutdownWG.Done()

	client := &http.Client{Timeout: 5 * time.Second}

	// Get initial JWK Set.
	err := tkr.updateJWKSet(client)
	if err != nil {
		glog.Warningf("Failed to get initial JWK Set: %s", err)
	}

	for {
		select {
		case <-tkr.shuttingDown:
			return

		case <-time.After(tkr.Config.JWKSetUpdateInterval.Duration):
			err = tkr.updateJWKSet(client)
			if err != nil {
				glog.Warningf("Failed to update JWK Set: %s", err)
			}
		}
	}
}

type jwkSet struct {
	Keys       []jose.JWK `json:"keys"`
	Issuer     string     `json:"issuer"`
	validUntil time.Time
}

func (tkr *Tracker) updateJWKSet(client *http.Client) error {
	glog.Info("Attemping to update JWK Set")
	resp, err := client.Get(tkr.Config.JWKSetURI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jwks jwkSet
	err = json.NewDecoder(resp.Body).Decode(&jwks)
	if err != nil {
		return err
	}

	if len(jwks.Keys) == 0 {
		return errors.New("Failed to find any keys from JWK Set URI")
	}

	if jwks.Issuer == "" {
		return errors.New("Failed to find any issuer from JWK Set URI")
	}

	ttl, _, _ := oidchttp.Cacheable(resp.Header)
	if ttl == 0 {
		ttl = jwkTTLFallback
	}
	jwks.validUntil = time.Now().Add(ttl)

	tkr.jwkSet = jwks
	glog.Info("Successfully updated JWK Set")
	return nil
}

func validateJWTSignature(jwt *jose.JWT, jwkSet *jwkSet) (bool, error) {
	for _, jwk := range jwkSet.Keys {
		v, err := jose.NewVerifier(jwk)
		if err != nil {
			return false, err
		}

		if err := v.Verify(jwt.Signature, []byte(jwt.Data())); err == nil {
			return true, nil
		}
	}

	return false, nil
}

func (tkr *Tracker) validateJWT(jwtStr, infohash string) error {
	jwkSet := tkr.jwkSet
	if time.Now().After(jwkSet.validUntil) {
		return fmt.Errorf("Failed verify JWT due to stale JWK Set")
	}

	jwt, err := jose.ParseJWT(jwtStr)
	if err != nil {
		return err
	}

	validated, err := validateJWTSignature(&jwt, &jwkSet)
	if err != nil {
		return err
	} else if !validated {
		return errors.New("Failed to verify JWT with all available verifiers")
	}

	claims, err := jwt.Claims()
	if err != nil {
		return err
	}

	if claimedIssuer, ok, err := claims.StringClaim("iss"); claimedIssuer != jwkSet.Issuer || err != nil || !ok {
		return errors.New("Failed to validate JWT issuer claim")
	}

	if claimedAudience, ok, err := claims.StringClaim("aud"); claimedAudience != tkr.Config.JWTAudience || err != nil || !ok {
		return errors.New("Failed to validate JWT audience claim")
	}

	claimedInfohash, ok, err := claims.StringClaim("infohash")
	if err != nil || !ok {
		return errors.New("Failed to validate JWT infohash claim")
	}

	unescapedInfohash, err := url.QueryUnescape(claimedInfohash)
	if err != nil {
		return errors.New("Failed to unescape JWT infohash claim")
	}

	if unescapedInfohash != infohash {
		return errors.New("Failed to match infohash claim with requested infohash")
	}

	return nil
}
