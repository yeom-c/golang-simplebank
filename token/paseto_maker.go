package token

import (
	"encoding/json"
	"time"

	"aidanwoods.dev/go-paseto"
)

type PasetoMaker struct {
	symmetricKey paseto.V4SymmetricKey
}

func NewPasetoMaker() (Maker, error) {
	key := paseto.NewV4SymmetricKey()

	return &PasetoMaker{key}, nil
}

func (maker *PasetoMaker) CreateToken(username string, duration time.Duration) (string, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", err
	}

	claims, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	pasetoToken, err := paseto.NewTokenFromClaimsJSON(claims, nil)
	if err != nil {
		return "", err
	}
	pasetoToken.SetIssuer(payload.Issuer)
	pasetoToken.SetIssuedAt(payload.IssuedAt)
	pasetoToken.SetExpiration(payload.ExpiresAt)
	pasetoToken.SetJti(payload.ID.String())

	return pasetoToken.V4Encrypt(maker.symmetricKey, nil), nil
}

func (maker *PasetoMaker) VerifyToken(encryptToken string) (*Payload, error) {
	parser := paseto.NewParser()
	token, err := parser.ParseV4Local(maker.symmetricKey, encryptToken, nil)
	if err != nil {
		return nil, err
	}

	var payload Payload
	if err := json.Unmarshal(token.ClaimsJSON(), &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}
