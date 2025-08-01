package core

import (
	"backend/database"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestUserPaginationToken_RoundTrip(t *testing.T) {
	cases := []UserPagination{
		{
			OrderBy:  nil,
			Filter:   "",
			LastUser: nil,
			Limit:    0,
		},
		{
			OrderBy:  nil,
			Filter:   "foo",
			LastUser: nil,
			Limit:    0,
		},
		{
			OrderBy: []database.UserOrder{
				{Field: database.UserFtLogin, Order: database.Asc},
			},
			Filter:   "",
			LastUser: nil,
			Limit:    0,
		},
		{
			OrderBy:  nil,
			Filter:   "",
			LastUser: nil,
			Limit:    42,
		},
		{

			OrderBy: nil,
			Filter:  "",
			LastUser: &database.User{
				ID:        "user_01",
				FtLogin:   "alice",
				FtID:      123,
				FtIsStaff: true,
				PhotoURL:  "http://example.com",
				LastSeen:  time.Date(2025, 5, 19, 12, 0, 0, 0, time.UTC),
				IsStaff:   false,
			},
			Limit: 0,
		},
		{
			OrderBy: []database.UserOrder{
				{Field: database.UserLastSeen, Order: database.Desc},
				{Field: database.UserID, Order: database.Asc},
			},
			Filter: "bar",
			LastUser: &database.User{
				ID:        "user_42",
				FtLogin:   "bob",
				FtID:      4242,
				FtIsStaff: false,
				PhotoURL:  "https://example.org/bob",
				LastSeen:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				IsStaff:   true,
			},
			Limit: 7,
		},
	}

	for _, orig := range cases {
		b64, err := EncodeUserPaginationToken(orig)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}
		decoded, err := DecodeUserPaginationToken(b64)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if diff := cmp.Diff(orig, decoded); diff != "" {
			t.Errorf("round-trip mismatch (-orig +decoded):\n%s", diff)
		}
	}
}

func TestEncodeUserPaginationToken(t *testing.T) {
	empty := UserPagination{}
	wantEmpty := "eyJPcmRlckJ5IjpudWxsLCJGaWx0ZXIiOiIiLCJMYXN0VXNlciI6bnVsbCwiTGltaXQiOjB9"
	// pre-computed base64 of {"OrderBy":[],"Filter":"","LastUser":null,"Limit":0}
	got, err := EncodeUserPaginationToken(empty)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	if got != wantEmpty {
		t.Errorf("encode(empty) = %q, want %q", got, wantEmpty)
	}
}

func TestDecodeUserPaginationToken(t *testing.T) {
	const b64 = "eyJPcmRlckJ5IjpudWxsLCJGaWx0ZXIiOiIiLCJMYXN0VXNlciI6bnVsbCwiTGltaXQiOjB9"
	want := UserPagination{}
	got, err := DecodeUserPaginationToken(b64)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("decode(empty) mismatch (-want +got):\n%s", diff)
	}

	// invalid base64 should error
	if _, err := DecodeUserPaginationToken("not-a-base64"); err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}
