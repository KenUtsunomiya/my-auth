package oauth2

import (
	"crypto/rand"
	"encoding/base64"
)

// NewState Generate a random state string for CSRF protection
func NewState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
