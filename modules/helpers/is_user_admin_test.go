package helpers

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestIsAdminMember(t *testing.T) {
	cases := []struct {
		name string
		typ  models.ChatMemberType
		want bool
	}{
		{"owner", models.ChatMemberTypeOwner, true},
		{"administrator", models.ChatMemberTypeAdministrator, true},
		{"member", models.ChatMemberTypeMember, false},
		{"restricted", models.ChatMemberTypeRestricted, false},
		{"left", models.ChatMemberTypeLeft, false},
		{"banned", models.ChatMemberTypeBanned, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isAdminMember(c.typ); got != c.want {
				t.Errorf("isAdminMember(%q) = %v, want %v", c.typ, got, c.want)
			}
		})
	}
}
