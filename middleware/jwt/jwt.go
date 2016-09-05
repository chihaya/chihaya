// Package jwt implements a Hook that fails an Announce if the client's request
// is missing a valid JSON Web Token.
//
// JWTs are validated against the standard claims in RFC7519 along with an
// extra "infohash" claim that verifies the client has access to the Swarm.
// RS256 keys are asychronously rotated from a provided JWK Set HTTP endpoint.
package jwt

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	jc "github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	log "github.com/Sirupsen/logrus"
	"github.com/mendsley/gojwk"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
)

var (
	// ErrMissingJWT is returned when a JWT is missing from a request.
	ErrMissingJWT = bittorrent.ClientError("unapproved request: missing jwt")

	// ErrInvalidJWT is returned when a JWT fails to verify.
	ErrInvalidJWT = bittorrent.ClientError("unapproved request: invalid jwt")
)

// Config represents all the values required by this middleware to fetch JWKs
// and verify JWTs.
type Config struct {
	Issuer            string        `yaml:"issuer"`
	Audience          string        `yaml:"audience"`
	JWKSetURL         string        `yaml:"jwk_set_url"`
	JWKUpdateInterval time.Duration `yaml:"jwk_set_update_interval"`
}

type hook struct {
	cfg        Config
	publicKeys map[string]crypto.PublicKey
	closing    chan struct{}
}

// NewHook returns an instance of the JWT middleware.
func NewHook(cfg Config) middleware.Hook {
	h := &hook{
		cfg:        cfg,
		publicKeys: map[string]crypto.PublicKey{},
		closing:    make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-h.closing:
				return
			case <-time.After(cfg.JWKUpdateInterval):
				resp, err := http.Get(cfg.JWKSetURL)
				if err != nil {
					log.Errorln("failed to fetch JWK Set: " + err.Error())
					continue
				}

				parsedJWKs := map[string]gojwk.Key{}
				err = json.NewDecoder(resp.Body).Decode(&parsedJWKs)
				if err != nil {
					resp.Body.Close()
					log.Errorln("failed to decode JWK JSON: " + err.Error())
					continue
				}
				resp.Body.Close()

				keys := map[string]crypto.PublicKey{}
				for kid, parsedJWK := range parsedJWKs {
					publicKey, err := parsedJWK.DecodePublicKey()
					if err != nil {
						log.Errorln("failed to decode JWK into public key: " + err.Error())
						continue
					}
					keys[kid] = publicKey
				}
				h.publicKeys = keys
			}
		}
	}()

	return h
}

func (h *hook) Stop() {
	close(h.closing)
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) error {
	if req.Params == nil {
		return ErrMissingJWT
	}

	jwtParam, ok := req.Params.String("jwt")
	if !ok {
		return ErrMissingJWT
	}

	if err := validateJWT(req.InfoHash, []byte(jwtParam), h.cfg.Issuer, h.cfg.Audience, h.publicKeys); err != nil {
		return ErrInvalidJWT
	}

	return nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) error {
	// Scrapes don't require any protection.
	return nil
}

func validateJWT(ih bittorrent.InfoHash, jwtBytes []byte, cfgIss, cfgAud string, publicKeys map[string]crypto.PublicKey) error {
	parsedJWT, err := jws.ParseJWT(jwtBytes)
	if err != nil {
		return err
	}

	claims := parsedJWT.Claims()
	if iss, ok := claims.Issuer(); !ok || iss != cfgIss {
		return jwt.ErrInvalidISSClaim
	}

	if aud, ok := claims.Audience(); !ok || !validAudience(aud, cfgAud) {
		return jwt.ErrInvalidAUDClaim
	}

	if ihClaim, ok := claims.Get("infohash").(string); !ok || !validInfoHash(ihClaim, ih) {
		return errors.New("claim \"infohash\" is invalid")
	}

	parsedJWS := parsedJWT.(jws.JWS)
	kid, ok := parsedJWS.Protected().Get("kid").(string)
	if !ok {
		return errors.New("invalid kid")
	}
	publicKey, ok := publicKeys[kid]
	if !ok {
		return errors.New("signed by unknown kid")
	}

	return parsedJWS.Verify(publicKey, jc.SigningMethodRS256)
}

func validAudience(aud []string, cfgAud string) bool {
	for _, a := range aud {
		if a == cfgAud {
			return true
		}
	}
	return false
}

func validInfoHash(claim string, ih bittorrent.InfoHash) bool {
	if len(claim) == 20 && bittorrent.InfoHashFromString(claim) == ih {
		return true
	}

	unescapedClaim, err := url.QueryUnescape(claim)
	if err != nil {
		return false
	}

	if len(unescapedClaim) == 20 && bittorrent.InfoHashFromString(unescapedClaim) == ih {
		return true
	}

	return false
}
