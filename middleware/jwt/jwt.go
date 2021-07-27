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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	jc "github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	"github.com/jzelinskie/stringz"
	"github.com/mendsley/gojwk"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/stop"
)

// Name is the name by which this middleware is registered with Chihaya.
const Name = "jwt"

func init() {
	middleware.RegisterDriver(Name, driver{})
}

var _ middleware.Driver = driver{}

type driver struct{}

func (d driver) NewHook(optionBytes []byte) (middleware.Hook, error) {
	var cfg Config
	err := yaml.Unmarshal(optionBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid options for middleware %s: %s", Name, err)
	}

	return NewHook(cfg)
}

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

func (cfg Config) MarshalZerologObject(e *zerolog.Event) {
	e.Str("issuer", cfg.Issuer).
		Str("audience", cfg.Audience).
		Str("jwkSetURL", cfg.JWKSetURL).
		Stringer("jwkUpdateInterval", cfg.JWKUpdateInterval)
}

type hook struct {
	cfg        Config
	publicKeys map[string]crypto.PublicKey
	closing    chan struct{}
}

// NewHook returns an instance of the JWT middleware.
func NewHook(cfg Config) (middleware.Hook, error) {
	log.Debug().EmbedObject(cfg).Msg("creating new JWT middleware")
	h := &hook{
		cfg:        cfg,
		publicKeys: map[string]crypto.PublicKey{},
		closing:    make(chan struct{}),
	}

	log.Debug().Msg("performing initial fetch of JWKs")
	err := h.updateKeys()
	if err != nil {
		return nil, errors.New("failed to fetch initial JWK Set: " + err.Error())
	}

	go func() {
		for {
			select {
			case <-h.closing:
				return
			case <-time.After(cfg.JWKUpdateInterval):
				log.Debug().Msg("performing fetch of JWKs")
				h.updateKeys()
			}
		}
	}()

	return h, nil
}

func (h *hook) updateKeys() error {
	resp, err := http.Get(h.cfg.JWKSetURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch JWK Set")
		return err
	}

	var parsedJWKs gojwk.Key
	err = json.NewDecoder(resp.Body).Decode(&parsedJWKs)
	if err != nil {
		resp.Body.Close()
		log.Error().Err(err).Msg("failed to decode JWK JSON")
		return err
	}
	resp.Body.Close()

	keys := map[string]crypto.PublicKey{}
	for _, parsedJWK := range parsedJWKs.Keys {
		publicKey, err := parsedJWK.DecodePublicKey()
		if err != nil {
			log.Error().Err(err).Msg("failed to decode JWK into public key")
			return err
		}
		keys[parsedJWK.Kid] = publicKey
	}
	h.publicKeys = keys

	log.Debug().Msg("successfully fetched JWK Set")
	return nil
}

func (h *hook) Stop() stop.Result {
	log.Debug().Msg("attempting to shutdown JWT middleware")
	select {
	case <-h.closing:
		return stop.AlreadyStopped
	default:
	}
	c := make(stop.Channel)
	go func() {
		close(h.closing)
		c.Done()
	}()
	return c.Result()
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	if req.Params == nil {
		return ctx, ErrMissingJWT
	}

	jwtParam, ok := req.Params.String("jwt")
	if !ok {
		return ctx, ErrMissingJWT
	}

	if err := validateJWT(req.InfoHash, []byte(jwtParam), h.cfg.Issuer, h.cfg.Audience, h.publicKeys); err != nil {
		return ctx, ErrInvalidJWT
	}

	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes don't require any protection.
	return ctx, nil
}

func validateJWT(ih bittorrent.InfoHash, jwtBytes []byte, cfgIss, cfgAud string, publicKeys map[string]crypto.PublicKey) error {
	parsedJWT, err := jws.ParseJWT(jwtBytes)
	if err != nil {
		return err
	}

	claims := parsedJWT.Claims()
	if iss, ok := claims.Issuer(); !ok || iss != cfgIss {
		log.Debug().
			Bool("exists", ok).
			Str("claim", iss).
			Str("config", cfgIss).
			Msg("unequal or missing issuer when validating JWT")
		return jwt.ErrInvalidISSClaim
	}

	if auds, ok := claims.Audience(); !ok || !stringz.SliceContains(auds, cfgAud) {
		log.Debug().
			Bool("exists", ok).
			Str("claim", strings.Join(auds, ",")).
			Str("config", cfgAud).
			Msg("unequal or missing audience when validating JWT")
		return jwt.ErrInvalidAUDClaim
	}

	ihHex := hex.EncodeToString(ih[:])
	if ihClaim, ok := claims.Get("infohash").(string); !ok || ihClaim != ihHex {
		log.Debug().
			Bool("exists", ok).
			Str("claim", ihClaim).
			Str("request", ihHex).
			Msg("unequal or missing infohash when validating JWT")
		return errors.New("claim \"infohash\" is invalid")
	}

	parsedJWS := parsedJWT.(jws.JWS)
	kid, ok := parsedJWS.Protected().Get("kid").(string)
	if !ok {
		log.Debug().
			Bool("exists", ok).
			Str("claim", kid).
			Msg("missing kid when validating JWT")
		return errors.New("invalid kid")
	}
	publicKey, ok := publicKeys[kid]
	if !ok {
		log.Debug().Str("kid", kid).Msg("missing public key forkid when validating JWT")
		return errors.New("signed by unknown kid")
	}

	err = parsedJWS.Verify(publicKey, jc.SigningMethodRS256)
	if err != nil {
		log.Debug().Err(err).Msg("failed to verify signature of JWT")
		return err
	}

	return nil
}
