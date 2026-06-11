package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"sync"
)

const cookieName = "gtc_session"

type sessionStore struct {
	mu     sync.Mutex
	tokens map[string]bool
}

func newSessionStore() *sessionStore {
	return &sessionStore{tokens: make(map[string]bool)}
}

func (s *sessionStore) create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	s.mu.Lock()
	s.tokens[token] = true
	s.mu.Unlock()
	return token, nil
}

func (s *sessionStore) valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens[token]
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.tokens, token)
	s.mu.Unlock()
}

// checkCredentials compares using constant-time to avoid timing attacks.
func checkCredentials(wantUser, wantPass, gotUser, gotPass string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(wantUser), []byte(gotUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(wantPass), []byte(gotPass)) == 1
	return userOK && passOK
}
