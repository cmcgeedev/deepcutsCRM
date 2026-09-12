// Package auth owns users, credentials, sessions and the realm middleware.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

const (
	RealmOffice   = "office"
	RealmDriver   = "driver"
	RealmCustomer = "customer"
)

type Auth struct {
	Q          *queries.Queries
	Now        func() time.Time
	Limiter    *Limiter
	SessionTTL time.Duration
}

func New(q *queries.Queries) *Auth {
	a := &Auth{Q: q, Now: time.Now, SessionTTL: 30 * 24 * time.Hour}
	a.Limiter = NewLimiter(10, 15*time.Minute, func() time.Time { return a.Now() })
	return a
}

type Session struct {
	ID          string
	UserID      int64
	Realm       string
	DisplayName string
	ExpiresAt   time.Time
}

var pinRe = regexp.MustCompile(`^[0-9]{6}$`)

func ValidPIN(p string) bool { return pinRe.MatchString(p) }

var ErrBadCredentials = service.Unauthorized("invalid credentials")
var ErrRateLimited = &service.Error{Status: 429, Code: "rate_limited", Message: "too many attempts, try again later"}

// dummyHash lets a failed lookup still pay for a bcrypt comparison, so a
// nonexistent user/PIN and a wrong password/PIN cost the same amount of time.
var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

func compareDummy(secret string) {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
	})
	bcrypt.CompareHashAndPassword(dummyHash, []byte(secret))
}

func (a *Auth) CreateOfficeUser(ctx context.Context, email, displayName, password string) (queries.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") {
		return queries.User{}, service.Invalid(map[string]string{"email": "must be an email address"})
	}
	if len(password) < 8 {
		return queries.User{}, service.Invalid(map[string]string{"password": "must be at least 8 characters"})
	}
	if len(password) > 72 {
		return queries.User{}, service.Invalid(map[string]string{"password": "must be at most 72 bytes"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return queries.User{}, err
	}
	u, err := a.Q.CreateUser(ctx, queries.CreateUserParams{
		Realm: RealmOffice, DisplayName: displayName, Email: sql.NullString{String: email, Valid: true},
		PasswordHash: sql.NullString{String: string(hash), Valid: true}, CreatedAt: a.Now().UTC(),
	})
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return queries.User{}, service.Conflict("duplicate", "an office user with that email already exists")
	}
	return u, err
}

func (a *Auth) CreateDriver(ctx context.Context, displayName, pin string) (queries.User, error) {
	if !ValidPIN(pin) {
		return queries.User{}, service.Invalid(map[string]string{"pin": "must be exactly 6 digits"})
	}
	if strings.TrimSpace(displayName) == "" {
		return queries.User{}, service.Invalid(map[string]string{"displayName": "required"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return queries.User{}, err
	}
	u, err := a.Q.CreateUser(ctx, queries.CreateUserParams{
		Realm: RealmDriver, DisplayName: strings.TrimSpace(displayName), PinHash: sql.NullString{String: string(hash), Valid: true}, CreatedAt: a.Now().UTC(),
	})
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return queries.User{}, service.Conflict("duplicate", "a driver with that name already exists")
	}
	return u, err
}

func (a *Auth) SetDriverPIN(ctx context.Context, displayName, pin string) error {
	if !ValidPIN(pin) {
		return service.Invalid(map[string]string{"pin": "must be exactly 6 digits"})
	}
	u, err := a.Q.GetDriverByName(ctx, displayName)
	if err != nil {
		return service.NotFound("driver")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := a.Q.SetUserPinHash(ctx, queries.SetUserPinHashParams{PinHash: sql.NullString{String: string(hash), Valid: true}, ID: u.ID}); err != nil {
		return err
	}
	// A new PIN invalidates any session created under the old one.
	return a.Q.DeleteUserSessions(ctx, u.ID)
}

func (a *Auth) DeactivateUser(ctx context.Context, emailOrName string) error {
	u, err := a.Q.GetUserByEmail(ctx, sql.NullString{String: strings.ToLower(emailOrName), Valid: true})
	if err != nil {
		u, err = a.Q.GetDriverByName(ctx, emailOrName)
		if err != nil {
			return service.NotFound("user")
		}
	}
	return a.Q.SetUserActive(ctx, queries.SetUserActiveParams{Active: false, ID: u.ID})
}

func (a *Auth) LoginOffice(ctx context.Context, email, password, ip string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 {
		return Session{}, ErrBadCredentials
	}
	if !a.Limiter.Allow("ip:"+ip) || !a.Limiter.Allow("email:"+email) {
		return Session{}, ErrRateLimited
	}
	u, err := a.Q.GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil || !u.Active || !u.PasswordHash.Valid {
		compareDummy(password)
		return Session{}, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash.String), []byte(password)) != nil {
		return Session{}, ErrBadCredentials
	}
	a.Limiter.Reset("email:" + email)
	a.Limiter.Reset("ip:" + ip) // successes from a shared warehouse IP must not lock the next driver out
	return a.createSession(ctx, u)
}

func (a *Auth) LoginDriver(ctx context.Context, userID int64, pin, ip string) (Session, error) {
	if !a.Limiter.Allow("ip:" + ip) {
		return Session{}, ErrRateLimited
	}
	u, err := a.Q.GetUser(ctx, userID)
	if err != nil {
		compareDummy(pin)
		return Session{}, ErrBadCredentials
	}
	// Only mint/consume the per-user rate limit key once we know the user exists.
	userKey := fmt.Sprintf("user:%d", u.ID)
	if !a.Limiter.Allow(userKey) {
		return Session{}, ErrRateLimited
	}
	if u.Realm != RealmDriver || !u.Active || !u.PinHash.Valid {
		compareDummy(pin)
		return Session{}, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PinHash.String), []byte(pin)) != nil {
		return Session{}, ErrBadCredentials
	}
	a.Limiter.Reset(userKey)
	a.Limiter.Reset("ip:" + ip)
	return a.createSession(ctx, u)
}

func (a *Auth) createSession(ctx context.Context, u queries.User) (Session, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return Session{}, err
	}
	s := Session{ID: base64.RawURLEncoding.EncodeToString(buf), UserID: u.ID, Realm: u.Realm, DisplayName: u.DisplayName, ExpiresAt: a.Now().UTC().Add(a.SessionTTL)}
	if err := a.Q.CreateSession(ctx, queries.CreateSessionParams{ID: s.ID, UserID: s.UserID, Realm: s.Realm, ExpiresAt: s.ExpiresAt}); err != nil {
		return Session{}, err
	}
	return s, nil
}

func (a *Auth) Logout(ctx context.Context, sessionID string) error {
	return a.Q.DeleteSession(ctx, sessionID)
}

func (a *Auth) Lookup(ctx context.Context, sessionID string) (Session, bool) {
	if sessionID == "" {
		return Session{}, false
	}
	row, err := a.Q.GetSession(ctx, sessionID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return Session{}, false
		}
		return Session{}, false
	}
	if !row.Active || row.ExpiresAt.Before(a.Now().UTC()) {
		return Session{}, false
	}
	return Session{ID: row.ID, UserID: row.UserID, Realm: row.Realm, DisplayName: row.DisplayName, ExpiresAt: row.ExpiresAt}, true
}
