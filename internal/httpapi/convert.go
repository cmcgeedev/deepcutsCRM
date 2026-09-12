package httpapi

import (
	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
)

func toUser(s auth.Session) api.User {
	return api.User{Id: s.UserID, Realm: api.UserRealm(s.Realm), DisplayName: s.DisplayName}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func i64Ptr(v *int64) *int64 { return v }
