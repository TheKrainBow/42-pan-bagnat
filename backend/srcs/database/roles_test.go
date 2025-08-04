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
		wantIDs []string
		wantErr bool
	}{
		// ----- STRUCT TESTS -----
		{
			name: "STRUCT TEST: ALL DATA",
			args: args{
				orderBy:  nil,
				filter:   "",
				lastRole: nil,
				limit:    0,
			},
			want: []Role{
				{ID: "role_01HZXYZDE0420", Name: "Student", Color: "#000000"},
				{ID: "role_01HZXYZDE0430", Name: "ADM", Color: "#00FF00"},
				{ID: "role_01HZXYZDE0440", Name: "Pedago", Color: "#FF0000"},
				{ID: "role_01HZXYZDE0450", Name: "IT", Color: "#FF00FF"},
				{ID: "role_01HZXYZDE0460", Name: "Reviewer", Color: "#AAAAAA"},
				{ID: "role_01HZXYZDE0461", Name: "Editor", Color: "#BBBBBB"},
				{ID: "role_01HZXYZDE0462", Name: "Manager", Color: "#CCCCCC"},
				{ID: "role_01HZXYZDE0463", Name: "Support", Color: "#DDDDDD"},
				{ID: "role_01HZXYZDE0464", Name: "Operator", Color: "#EEEEEE"},
				{ID: "role_01HZXYZDE0465", Name: "Guest", Color: "#111111"},
				{ID: "role_01HZXYZDE0466", Name: "Developer", Color: "#222222"},
				{ID: "role_01HZXYZDE0467", Name: "Analyst", Color: "#333333"},
				{ID: "role_01HZXYZDE0468", Name: "Designer", Color: "#444444"},
				{ID: "role_01HZXYZDE0469", Name: "Tester", Color: "#555555"},
				{ID: "role_01HZXYZDE0470", Name: "Maintainer", Color: "#666666"},
				{ID: "role_01HZXYZDE0471", Name: "Contributor", Color: "#777777"},
				{ID: "role_01HZXYZDE0472", Name: "Owner", Color: "#888888"},
				{ID: "role_01HZXYZDE0473", Name: "SuperAdmin", Color: "#999999"},
				{ID: "role_01HZXYZDE0474", Name: "Moderator", Color: "#123456"},
				{ID: "role_01HZXYZDE0475", Name: "Auditor", Color: "#654321"},
				{ID: "role_01HZXYZDE0476", Name: "Architect", Color: "#ABCDEF"},
				{ID: "role_01HZXYZDE0477", Name: "Coordinator", Color: "#FEDCBA"},
				{ID: "role_01HZXYZDE0478", Name: "Planner", Color: "#0F0F0F"},
				{ID: "role_01HZXYZDE0479", Name: "Strategist", Color: "#F0F0F0"},
			},
			wantErr: false,
		},
		// ----- LIMIT TESTS -----
		{
			name: "LIMIT TESTS: 1",
			args: args{
				orderBy:  nil,
				filter:   "",
				lastRole: nil,
				limit:    1,
			},
			wantIDs: []string{
				"role_01HZXYZDE0420",
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
			wantIDs: []string{
				"role_01HZXYZDE0420",
				"role_01HZXYZDE0430",
				"role_01HZXYZDE0440",
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
			wantIDs: []string{
				"role_01HZXYZDE0430",
				"role_01HZXYZDE0467",
				"role_01HZXYZDE0476",
				"role_01HZXYZDE0475",
				"role_01HZXYZDE0471",
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
			wantIDs: []string{
				"role_01HZXYZDE0469",
				"role_01HZXYZDE0463",
				"role_01HZXYZDE0473",
				"role_01HZXYZDE0420",
				"role_01HZXYZDE0479",
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
			wantIDs: []string{
				"role_01HZXYZDE0475",
				"role_01HZXYZDE0471",
				"role_01HZXYZDE0477",
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
				lastRole: &Role{ID: "role_01HZXYZDE0476", Name: "Architect", Color: "#ABCDEF"},
				limit:    3,
			},
			wantIDs: []string{
				"role_01HZXYZDE0475",
				"role_01HZXYZDE0471",
				"role_01HZXYZDE0477",
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

			if tt.want != nil {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("GetAllRoles() mismatch (-want +got):\n%s", diff)
					t.Errorf("want:\n%s", formatRoles(tt.want))
					t.Errorf(" got:\n%s", formatRoles(got))
				}
				return
			}

			var gotIDs []string
			for _, m := range got {
				gotIDs = append(gotIDs, m.ID)
			}
			if diff := cmp.Diff(tt.wantIDs, gotIDs); diff != "" {
				t.Errorf("IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
