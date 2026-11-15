package dashboard

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"math/big"
	"slices"
	"strings"
)

// verifyIDToken checks an id_token end to end: shape, RS256 signature
// via the issuer's JWKS, and the claim contract (issuer byte-equality,
// audience, expiry with skew, nonce binding).
func (o *oidcLogin) verifyIDToken(
	ctx context.Context,
	raw, expectedNonce string,
) (oidcClaims, error) {
	header, payload, signature, signed, err := splitJWT(raw)
	if err != nil {
		return oidcClaims{}, err
	}

	if header.Algorithm != oidcJWTAlgorithm {
		return oidcClaims{}, fmt.Errorf("%w: %q", errOIDCUnsupportedAlg, header.Algorithm)
	}

	key, err := o.jwksFor(ctx, header.KeyID)
	if err != nil {
		return oidcClaims{}, err
	}

	digest := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return oidcClaims{}, fmt.Errorf("%w: %w", errOIDCSignature, err)
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
		return jwtHeader{}, nil, nil, nil, errOIDCNotAJWS
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
		return fmt.Errorf("%w: %q vs %q", errOIDCIssuerMismatch, claims.Issuer, o.cfg.Issuer)
	}

	if !claims.Audience.has(o.cfg.ClientID) {
		return fmt.Errorf("%w: %q", errOIDCAudience, o.cfg.ClientID)
	}

	if claims.Expiry <= o.clock().Add(-oidcClockSkew).Unix() {
		return errOIDCTokenExpired
	}

	if expectedNonce == "" || claims.Nonce != expectedNonce {
		return errOIDCNonceMismatch
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
			return fmt.Errorf("decoding the string audience: %w", err)
		}

		*a = oidcAudience{single}

		return nil
	}

	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return fmt.Errorf("decoding the array audience: %w", err)
	}

	*a = many

	return nil
}

func (a oidcAudience) has(clientID string) bool {
	return slices.Contains(a, clientID)
}

// oidcClaims is the subset of the id_token/userinfo payload the login
// reads.
type oidcClaims struct {
	Issuer        string       `json:"iss"`
	Subject       string       `json:"sub"`
	Audience      oidcAudience `json:"aud"`
	Expiry        int64        `json:"exp"`
	Nonce         string       `json:"nonce"`
	Email         string       `json:"email"`
	EmailVerified bool         `json:"email_verified"`
	Groups        []string     `json:"groups"`
	PreferredName string       `json:"preferred_username"`
}

func (c oidcClaims) hasGroup(group string) bool {
	return slices.Contains(c.Groups, group)
}

// mergeClaims fills only the MISSING profile claims from a userinfo
// fallback — present token claims always win.
func mergeClaims(claims, fallback oidcClaims) oidcClaims {
	if claims.Email == "" {
		claims.Email = fallback.Email
	}

	if !claims.EmailVerified && fallback.EmailVerified {
		claims.EmailVerified = fallback.EmailVerified
	}

	if claims.Groups == nil {
		claims.Groups = fallback.Groups
	}

	if claims.PreferredName == "" {
		claims.PreferredName = fallback.PreferredName
	}

	return claims
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
		return nil, fmt.Errorf("%w: %d vs %d", errOIDCWeakModulus, key.N.BitLen(), oidcMinRSABits)
	}

	return key, nil
}

// oidcMinRSABits rejects toy keys an issuer should never publish.
const oidcMinRSABits = 2048

func decodeSegment(segment string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return nil, fmt.Errorf("decoding a base64url segment: %w", err)
	}

	return decoded, nil
}
