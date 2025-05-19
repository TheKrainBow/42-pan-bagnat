package core

import (
	"backend/database"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRolePaginationToken_RoundTrip(t *testing.T) {
	cases := []RolePagination{
		{
			OrderBy:  nil,
			Filter:   "",
			LastRole: nil,
			Limit:    0,
		},
		{
			OrderBy:  nil,
			Filter:   "foo",
			LastRole: nil,
			Limit:    0,
		},
		{
			OrderBy: []database.RoleOrder{
				{Field: database.RoleName, Order: database.Asc},
			},
			Filter:   "",
			LastRole: nil,
			Limit:    0,
		},
		{
			OrderBy:  nil,
			Filter:   "",
			LastRole: nil,
			Limit:    42,
		},
		{

			OrderBy: nil,
			Filter:  "",
			LastRole: &database.Role{
				ID:    "user_01",
				Name:  "alice",
				Color: "0xFFFFFF",
			},
			Limit: 0,
		},
		{
			OrderBy: []database.RoleOrder{
				{Field: database.RoleName, Order: database.Desc},
				{Field: database.RoleID, Order: database.Asc},
			},
			Filter: "bar",
			LastRole: &database.Role{
				ID:    "user_01",
				Name:  "alice",
				Color: "0xFFFFFF",
			},
			Limit: 7,
		},
	}

	for _, orig := range cases {
		b64, err := EncodeRolePaginationToken(orig)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}
		decoded, err := DecodeRolePaginationToken(b64)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if diff := cmp.Diff(orig, decoded); diff != "" {
			t.Errorf("round-trip mismatch (-orig +decoded):\n%s", diff)
		}
	}
}

func TestEncodeRolePaginationToken(t *testing.T) {
	empty := RolePagination{}
	wantEmpty := "eyJPcmRlckJ5IjpudWxsLCJGaWx0ZXIiOiIiLCJMYXN0Um9sZSI6bnVsbCwiTGltaXQiOjB9"
	// pre-computed base64 of {"OrderBy":[],"Filter":"","LastRole":null,"Limit":0}
	got, err := EncodeRolePaginationToken(empty)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	if got != wantEmpty {
		t.Errorf("encode(empty) = %q, want %q", got, wantEmpty)
	}
}

func TestDecodeRolePaginationToken(t *testing.T) {
	const b64 = "eyJPcmRlckJ5IjpudWxsLCJGaWx0ZXIiOiIiLCJMYXN0Um9sZSI6bnVsbCwiTGltaXQiOjB9"
	want := RolePagination{}
	got, err := DecodeRolePaginationToken(b64)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("decode(empty) mismatch (-want +got):\n%s", diff)
	}

	// invalid base64 should error
	if _, err := DecodeRolePaginationToken("not-a-base64"); err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}
