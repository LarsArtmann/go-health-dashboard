package dashboard

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// verifyIDToken checks an id_token end to end: shape, RS256 signature
// via the issuer's JWKS, and the claim contract (issuer byte-equality,
// audience, expiry with skew, nonce binding).
func (o *oidcLogin) verifyIDToken(ctx context.Context, raw, expectedNonce string) (oidcClaims, error) {
	header, payload, signature, signed, err := splitJWT(raw)
	if err != nil {
		return oidcClaims{}, err
	}

	if header.Algorithm != oidcJWTAlgorithm {
		return oidcClaims{}, fmt.Errorf("unsupported id_token algorithm %q (only RS256)", header.Algorithm)
	}

	key, err := o.jwksFor(ctx, header.KeyID)
	if err != nil {
		return oidcClaims{}, err
	}

	digest := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return oidcClaims{}, errors.New("signature does not verify")
	}

	claims, err := decodeClaims(payload)
	if err != nil {
		return oidcClaims{}, err
	}

	return claims, o.checkClaims(claims, expectedNonce)
}

// oidcJWTAlgorithm is the only accepted id_token algorithm.
const oidcJWTAlgorithm = "RS256"

// jwtParts is the number of dot-separated segments in a JWS.
const jwtParts = 3

type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

// splitJWT decodes the three base64url segments of a JWS and returns
// the signed bytes (header.payload) alongside.
func splitJWT(raw string) (jwtHeader, []byte, []byte, []byte, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != jwtParts {
		return jwtHeader{}, nil, nil, nil, errors.New("id_token is not a three-part JWS")
	}

	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return jwtHeader{}, nil, nil, nil, fmt.Errorf("decoding the id_token header: %w", err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtHeader{}, nil, nil, nil, fmt.Errorf("decoding the id_token header: %w", err)
	}

	payload, err := decodeSegment(parts[1])
	if err != nil {
		return jwtHeader{}, nil, nil, nil, fmt.Errorf("decoding the id_token payload: %w", err)
	}

	signature, err := decodeSegment(parts[2])
	if err != nil {
		return jwtHeader{}, nil, nil, nil, fmt.Errorf("decoding the id_token signature: %w", err)
	}

	signed := []byte(parts[0] + "." + parts[1])

	return header, payload, signature, signed, nil
}

// checkClaims enforces the claim contract against the middleware's own
// configuration and the login attempt.
func (o *oidcLogin) checkClaims(claims oidcClaims, expectedNonce string) error {
	if claims.Issuer != o.cfg.Issuer {
		return fmt.Errorf("id_token issuer %q does not match %q", claims.Issuer, o.cfg.Issuer)
	}

	if !claims.audienceHas(o.cfg.ClientID) {
		return fmt.Errorf("id_token audience does not include client %q", o.cfg.ClientID)
	}

	if claims.Expiry <= o.clock().Add(-oidcClockSkew).Unix() {
		return errors.New("id_token is expired")
	}

	if expectedNonce == "" || claims.Nonce != expectedNonce {
		return errors.New("id_token nonce does not match the login attempt")
	}

	return nil
}

// oidcAudience models the aud claim, which the OIDC core allows as a
// string or an array of strings.
type oidcAudience []string

func (a *oidcAudience) UnmarshalJSON(raw []byte) error {
	if len(raw) > 0 && raw[0] == '"' {
		var single string
		if err := json.Unmarshal(raw, &single); err != nil {
			return err
		}

		*a = oidcAudience{single}

		return nil
	}

	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return err
	}

	*a = many

	return nil
}

func (a oidcAudience) has(clientID string) bool {
	for _, aud := range a {
		if aud == clientID {
			return true
		}
	}

	return false
}

// oidcClaims is the subset of the id_token/userinfo payload the login
// reads.
type oidcClaims struct {
	Issuer          string        `json:"iss"`
	Subject         string        `json:"sub"`
	Audience        oidcAudience  `json:"aud"`
	Expiry          int64         `json:"exp"`
	Nonce           string        `json:"nonce"`
	Email           string        `json:"email"`
	EmailVerified   bool          `json:"email_verified"`
	Groups          []string      `json:"groups"`
	PreferredName   string        `json:"preferred_username"`
}

func (c oidcClaims) audienceHas(clientID string) bool {
	return c.Audience.has(clientID)
}

func (c oidcClaims) hasGroup(group string) bool {
	for _, granted := range c.Groups {
		if granted == group {
			return true
		}
	}

	return false
}

// merge fills only the MISSING profile claims from a userinfo fallback
// — present token claims always win.
func (c *oidcClaims) merge(fallback oidcClaims) {
	if c.Email == "" {
		c.Email = fallback.Email
	}

	if !c.EmailVerified && fallback.EmailVerified {
		c.EmailVerified = fallback.EmailVerified
	}

	if c.Groups == nil {
		c.Groups = fallback.Groups
	}

	if c.PreferredName == "" {
		c.PreferredName = fallback.PreferredName
	}
}

func decodeClaims(payload []byte) (oidcClaims, error) {
	var claims oidcClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return oidcClaims{}, fmt.Errorf("decoding the id_token claims: %w", err)
	}

	return claims, nil
}

// oidcRSAKey builds an RSA public key from the JWK modulus/exponent
// pair (base64url, big-endian).
func oidcRSAKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := decodeSegment(modulus)
	if err != nil {
		return nil, fmt.Errorf("decoding the JWK modulus: %w", err)
	}

	eBytes, err := decodeSegment(exponent)
	if err != nil {
		return nil, fmt.Errorf("decoding the JWK exponent: %w", err)
	}

	key := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}

	if key.N.BitLen() < oidcMinRSABits {
		return nil, fmt.Errorf("JWK modulus is %d bits (minimum %d)", key.N.BitLen(), oidcMinRSABits)
	}

	return key, nil
}

// oidcMinRSABits rejects toy keys an issuer should never publish.
const oidcMinRSABits = 2048

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}
