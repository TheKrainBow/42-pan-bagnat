package core

import (
	"backend/database"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestModulePaginationToken_RoundTrip(t *testing.T) {
	cases := []ModulePagination{
		{
			OrderBy:    nil,
			Filter:     "",
			LastModule: nil,
			Limit:      0,
		},
		{
			OrderBy:    nil,
			Filter:     "foo",
			LastModule: nil,
			Limit:      0,
		},
		{
			OrderBy: []database.ModuleOrder{
				{Field: database.ModuleName, Order: database.Asc},
			},
			Filter:     "",
			LastModule: nil,
			Limit:      0,
		},
		{
			OrderBy:    nil,
			Filter:     "",
			LastModule: nil,
			Limit:      42,
		},
		{

			OrderBy: nil,
			Filter:  "",
			LastModule: &database.Module{
				ID:            "user_01",
				Name:          "alice",
				Version:       "123",
				Status:        "running",
				GitURL:        "http://example.com",
				LatestVersion: "124",
				LateCommits:   2,
			},
			Limit: 0,
		},
		{
			OrderBy: []database.ModuleOrder{
				{Field: database.ModuleName, Order: database.Desc},
				{Field: database.ModuleID, Order: database.Asc},
			},
			Filter: "bar",
			LastModule: &database.Module{
				ID:            "user_01",
				Name:          "alice",
				Version:       "123",
				Status:        "running",
				GitURL:        "http://example.com",
				LatestVersion: "124",
				LateCommits:   2,
			},
			Limit: 7,
		},
	}

	for _, orig := range cases {
		b64, err := EncodeModulePaginationToken(orig)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}
		decoded, err := DecodeModulePaginationToken(b64)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if diff := cmp.Diff(orig, decoded); diff != "" {
			t.Errorf("round-trip mismatch (-orig +decoded):\n%s", diff)
		}
	}
}

func TestEncodeModulePaginationToken(t *testing.T) {
	empty := ModulePagination{}
	wantEmpty := "eyJPcmRlckJ5IjpudWxsLCJGaWx0ZXIiOiIiLCJMYXN0TW9kdWxlIjpudWxsLCJMaW1pdCI6MH0="
	// pre-computed base64 of {"OrderBy":[],"Filter":"","LastModule":null,"Limit":0}
	got, err := EncodeModulePaginationToken(empty)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	if got != wantEmpty {
		t.Errorf("encode(empty) = %q, want %q", got, wantEmpty)
	}
}

func TestDecodeModulePaginationToken(t *testing.T) {
	const b64 = "eyJPcmRlckJ5IjpudWxsLCJGaWx0ZXIiOiIiLCJMYXN0TW9kdWxlIjpudWxsLCJMaW1pdCI6MH0="
	want := ModulePagination{}
	got, err := DecodeModulePaginationToken(b64)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("decode(empty) mismatch (-want +got):\n%s", diff)
	}

	// invalid base64 should error
	if _, err := DecodeModulePaginationToken("not-a-base64"); err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}
