package database

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
)

//go:embed test_data/test_data_roles.sql
var roleTestDataSQL string

func TestGetAllRoles(t *testing.T) {
	type args struct {
		orderBy  *[]RoleOrder
		filter   string
		lastRole *Role
		limit    int
	}
	tests := []struct {
		name    string
		args    args
		want    []Role
		wantErr bool
	}{
		// ----- LIMIT TESTS -----
		{
			name: "LIMIT TESTS: 1",
			args: args{
				orderBy:  nil,
				filter:   "",
				lastRole: nil,
				limit:    1,
			},
			want: []Role{
				{ID: "role_01HZXYZDE0420", Name: "Student", Color: "0x000000"},
			},
			wantErr: false,
		},
		{
			name: "LIMIT TESTS: 3",
			args: args{
				orderBy:  nil,
				filter:   "",
				lastRole: nil,
				limit:    3,
			},
			want: []Role{
				{ID: "role_01HZXYZDE0420", Name: "Student", Color: "0x000000"},
				{ID: "role_01HZXYZDE0430", Name: "ADM", Color: "0x00FF00"},
				{ID: "role_01HZXYZDE0440", Name: "Pedago", Color: "0xFF0000"},
			},
			wantErr: false,
		},

		// ----- ORDER TESTS (by Name) -----
		{
			name: "ORDER TEST: Asc on Name, limit 5",
			args: args{
				orderBy: &[]RoleOrder{
					{Field: RoleName, Order: Asc},
				},
				filter:   "",
				lastRole: nil,
				limit:    5,
			},
			want: []Role{
				{ID: "role_01HZXYZDE0430", Name: "ADM", Color: "0x00FF00"},
				{ID: "role_01HZXYZDE0467", Name: "Analyst", Color: "0x333333"},
				{ID: "role_01HZXYZDE0476", Name: "Architect", Color: "0xABCDEF"},
				{ID: "role_01HZXYZDE0475", Name: "Auditor", Color: "0x654321"},
				{ID: "role_01HZXYZDE0471", Name: "Contributor", Color: "0x777777"},
			},
			wantErr: false,
		},
		{
			name: "ORDER TEST: Desc on Name, limit 5",
			args: args{
				orderBy: &[]RoleOrder{
					{Field: RoleName, Order: Desc},
				},
				filter:   "",
				lastRole: nil,
				limit:    5,
			},
			want: []Role{
				{ID: "role_01HZXYZDE0469", Name: "Tester", Color: "0x555555"},
				{ID: "role_01HZXYZDE0463", Name: "Support", Color: "0xDDDDDD"},
				{ID: "role_01HZXYZDE0473", Name: "SuperAdmin", Color: "0x999999"},
				{ID: "role_01HZXYZDE0420", Name: "Student", Color: "0x000000"},
				{ID: "role_01HZXYZDE0479", Name: "Strategist", Color: "0xF0F0F0"},
			},
			wantErr: false,
		},

		// ----- FILTER TEST (by Name substring) -----
		{
			name: "FILTER TEST: filter \"or\", Name asc, limit 3",
			args: args{
				orderBy: &[]RoleOrder{
					{Field: RoleName, Order: Asc},
				},
				filter:   "or",
				lastRole: nil,
				limit:    3,
			},
			want: []Role{
				{ID: "role_01HZXYZDE0475", Name: "Auditor", Color: "0x654321"},
				{ID: "role_01HZXYZDE0471", Name: "Contributor", Color: "0x777777"},
				{ID: "role_01HZXYZDE0477", Name: "Coordinator", Color: "0xFEDCBA"},
			},
			wantErr: false,
		},

		// ----- PAGINATION TEST (Name asc, page size 3, page 2) -----
		{
			name: "PAGINATION: Page size 3, page number 2, Name asc",
			args: args{
				orderBy: &[]RoleOrder{
					{Field: RoleName, Order: Asc},
				},
				filter:   "",
				lastRole: &Role{ID: "role_01HZXYZDE0476", Name: "Architect", Color: "0xABCDEF"},
				limit:    3,
			},
			want: []Role{
				{ID: "role_01HZXYZDE0475", Name: "Auditor", Color: "0x654321"},
				{ID: "role_01HZXYZDE0471", Name: "Contributor", Color: "0x777777"},
				{ID: "role_01HZXYZDE0477", Name: "Coordinator", Color: "0xFEDCBA"},
			},
			wantErr: false,
		},
	}

	CreateAndPopulateDatabase(t, "test_get_roles_db", roleTestDataSQL)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAllRoles(tt.args.orderBy, tt.args.filter, tt.args.lastRole, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetAllRoles() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetAllRoles() mismatch (-want +got):\n%s", diff)
				t.Errorf("want:\n%s", formatRoles(tt.want))
				t.Errorf(" got:\n%s", formatRoles(got))
			}
		})
	}
}
