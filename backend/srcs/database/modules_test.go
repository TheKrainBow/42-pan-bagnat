package database

import (
	_ "embed"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

//go:embed test_data/test_data_modules.sql
var moduleTestDataSQL string

func TestGetAllModules(t *testing.T) {
	type args struct {
		orderBy    *[]ModuleOrder
		filter     string
		lastModule *Module
		limit      int
	}
	tests := []struct {
		name    string
		args    args
		want    []Module
		wantIDs []string
		wantErr bool
	}{
		{
			name: "STRUCTURE TEST: All data",
			args: args{orderBy: nil, filter: "", lastModule: nil, limit: -1},
			want: []Module{
				{ID: "module_01HZXYZDE0420", Name: "captain-hook", Version: "1.2", Status: "enabled", URL: "https://github.com/42nice/captain-hook", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.7", LateCommits: 5, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0430", Name: "adm-stud", Version: "1.5", Status: "enabled", URL: "https://github.com/42nice/adm-stud", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.5", LateCommits: 0, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0440", Name: "adm-manager", Version: "1.0", Status: "enabled", URL: "https://github.com/42nice/adm-manager", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.0", LateCommits: 0, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0450", Name: "student-info", Version: "1.8", Status: "enabled", URL: "https://github.com/42nice/student-info", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.9", LateCommits: 1, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0460", Name: "role-manager", Version: "1.0", Status: "enabled", URL: "https://github.com/42nice/role-manager", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.0", LateCommits: 0, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0461", Name: "role-editor", Version: "1.1", Status: "enabled", URL: "https://github.com/42nice/role-editor", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.1", LateCommits: 2, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0462", Name: "support-tool", Version: "2.0", Status: "enabled", URL: "https://github.com/42nice/support-tool", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "2.0", LateCommits: 1, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0463", Name: "analytics", Version: "3.2", Status: "enabled", URL: "https://github.com/42nice/analytics", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "3.2", LateCommits: 4, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0464", Name: "design-proto", Version: "0.9", Status: "enabled", URL: "https://github.com/42nice/design-proto", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "0.9", LateCommits: 0, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0465", Name: "test-suite", Version: "5.4", Status: "enabled", URL: "https://github.com/42nice/test-suite", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "5.4", LateCommits: 3, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0466", Name: "deploy-automate", Version: "1.5", Status: "enabled", URL: "https://github.com/42nice/deploy-automate", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.5", LateCommits: 0, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0467", Name: "strategy-dash", Version: "4.0", Status: "enabled", URL: "https://github.com/42nice/strategy-dash", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "4.0", LateCommits: 2, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0468", Name: "zorbi-app", Version: "1.0", Status: "disabled", URL: "https://example.com/zorbi", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.0", LateCommits: 0, LastUpdate: time.Date(2025, 4, 21, 8, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0469", Name: "alpha-tool", Version: "2.2", Status: "disabled", URL: "https://example.com/alpha", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "2.2", LateCommits: 1, LastUpdate: time.Date(2025, 4, 21, 8, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0470", Name: "beta-service", Version: "3.5", Status: "enabled", URL: "https://example.com/beta", IconeURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "3.5", LateCommits: 1, LastUpdate: time.Date(2025, 4, 21, 8, 0, 0, 0, time.UTC)},
			},
		},
		// ----- LIMIT TESTS -----
		{
			name: "LIMIT TEST: negative (no limit)",
			args: args{orderBy: nil, filter: "", lastModule: nil, limit: -1},
			wantIDs: []string{
				"module_01HZXYZDE0420",
				"module_01HZXYZDE0430",
				"module_01HZXYZDE0440",
				"module_01HZXYZDE0450",
				"module_01HZXYZDE0460",
				"module_01HZXYZDE0461",
				"module_01HZXYZDE0462",
				"module_01HZXYZDE0463",
				"module_01HZXYZDE0464",
				"module_01HZXYZDE0465",
				"module_01HZXYZDE0466",
				"module_01HZXYZDE0467",
				"module_01HZXYZDE0468",
				"module_01HZXYZDE0469",
				"module_01HZXYZDE0470",
			},
		},
		{
			name: "LIMIT TEST: zero (no limit)",
			args: args{orderBy: nil, filter: "", lastModule: nil, limit: 0},
			wantIDs: []string{
				"module_01HZXYZDE0420",
				"module_01HZXYZDE0430",
				"module_01HZXYZDE0440",
				"module_01HZXYZDE0450",
				"module_01HZXYZDE0460",
				"module_01HZXYZDE0461",
				"module_01HZXYZDE0462",
				"module_01HZXYZDE0463",
				"module_01HZXYZDE0464",
				"module_01HZXYZDE0465",
				"module_01HZXYZDE0466",
				"module_01HZXYZDE0467",
				"module_01HZXYZDE0468",
				"module_01HZXYZDE0469",
				"module_01HZXYZDE0470",
			},
		},
		{
			name: "LIMIT TEST: 1",
			args: args{orderBy: nil, filter: "", lastModule: nil, limit: 1},
			wantIDs: []string{
				"module_01HZXYZDE0420",
			},
		},
		{
			name: "LIMIT TEST: 5",
			args: args{orderBy: nil, filter: "", lastModule: nil, limit: 5},
			wantIDs: []string{
				"module_01HZXYZDE0420",
				"module_01HZXYZDE0430",
				"module_01HZXYZDE0440",
				"module_01HZXYZDE0450",
				"module_01HZXYZDE0460",
			},
		},
		// ----- ORDER TESTS -----
		{
			name: "ORDER TEST: Asc on ID, limit 3",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleID, Order: Asc}},
				filter:  "", lastModule: nil, limit: 3,
			},
			wantIDs: []string{
				"module_01HZXYZDE0420",
				"module_01HZXYZDE0430",
				"module_01HZXYZDE0440",
			},
		},
		{
			name: "ORDER TEST: Desc on ID, limit 3",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleID, Order: Desc}},
				filter:  "", lastModule: nil, limit: 3,
			},
			wantIDs: []string{
				"module_01HZXYZDE0470",
				"module_01HZXYZDE0469",
				"module_01HZXYZDE0468",
			},
		},
		{
			name: "ORDER TEST: Asc on Name, limit 4",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleName, Order: Asc}},
				filter:  "", lastModule: nil, limit: 4,
			},
			wantIDs: []string{
				"module_01HZXYZDE0440",
				"module_01HZXYZDE0430",
				"module_01HZXYZDE0469",
				"module_01HZXYZDE0463",
			},
		},
		{
			name: "ORDER TEST: Desc on Name, limit 4",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleName, Order: Desc}},
				filter:  "", lastModule: nil, limit: 4,
			},
			wantIDs: []string{
				"module_01HZXYZDE0468",
				"module_01HZXYZDE0465",
				"module_01HZXYZDE0462",
				"module_01HZXYZDE0450",
			},
		},
		{
			name: "ORDER TEST: Asc on LateCommits, limit 3",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleLateCommits, Order: Asc}},
				filter:  "", lastModule: nil, limit: 3,
			},
			wantIDs: []string{
				"module_01HZXYZDE0430",
				"module_01HZXYZDE0440",
				"module_01HZXYZDE0460",
			},
		},
		{
			name: "ORDER TEST: Desc on LateCommits, limit 3",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleLateCommits, Order: Desc}},
				filter:  "", lastModule: nil, limit: 3,
			},
			wantIDs: []string{
				"module_01HZXYZDE0420",
				"module_01HZXYZDE0463",
				"module_01HZXYZDE0465",
			},
		},
		{
			name: "ORDER TEST: Asc on LastUpdate, limit 2",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleLastUpdate, Order: Asc}},
				filter:  "", lastModule: nil, limit: 2,
			},
			wantIDs: []string{
				"module_01HZXYZDE0420",
				"module_01HZXYZDE0430",
			},
		},
		{
			name: "ORDER TEST: Desc on LastUpdate, limit 2",
			args: args{
				orderBy: &[]ModuleOrder{{Field: ModuleLastUpdate, Order: Desc}},
				filter:  "", lastModule: nil, limit: 2,
			},
			wantIDs: []string{
				"module_01HZXYZDE0470",
				"module_01HZXYZDE0469",
			},
		},
		{
			name: "FILTER TEST: match \"adm\", limit 3",
			args: args{orderBy: &[]ModuleOrder{{Field: ModuleName, Order: Asc}}, filter: "adm", lastModule: nil, limit: 3},
			wantIDs: []string{
				"module_01HZXYZDE0440",
				"module_01HZXYZDE0430",
			},
		},
		{
			name: "FILTER TEST: no match \"xyz\"",
			args: args{orderBy: nil, filter: "xyz", lastModule: nil, limit: 5},
			want: []Module{},
		},

		// ----- FILTER + PAGINATION -----
		{
			name: "FILTER+PAGINATION: \"adm\", page size 1, page 2",
			args: args{
				orderBy: &[]ModuleOrder{
					{Field: ModuleName, Order: Asc},
				},
				filter:     "adm",
				lastModule: nil,
				limit:      1,
			},
			wantIDs: []string{
				"module_01HZXYZDE0440",
			},
			wantErr: false,
		},
		{
			name: "FILTER+PAGINATION: \"adm\", page size 1, page 2",
			args: args{
				orderBy: &[]ModuleOrder{
					{Field: ModuleName, Order: Asc},
				},
				filter: "adm",
				lastModule: &Module{
					ID:            "module_01HZXYZDE0440",
					Name:          "adm-manager",
					Version:       "1.0",
					Status:        "enabled",
					URL:           "https://github.com/42nice/adm-manager",
					LatestVersion: "1.0",
					LateCommits:   0,
					LastUpdate:    time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC),
				},
				limit: 1,
			},
			wantIDs: []string{
				"module_01HZXYZDE0430",
			},
			wantErr: false,
		},

		// ----- PAGINATION EDGE -----
		{
			name: "PAGINATION EDGE: lastModule at end id asc",
			args: args{
				orderBy: nil,
				filter:  "",
				lastModule: &Module{
					ID:            "module_01HZXYZDE0470",
					Name:          "beta-service",
					Version:       "3.5",
					Status:        "enabled",
					URL:           "https://example.com/beta",
					LatestVersion: "3.5",
					LateCommits:   1,
					LastUpdate:    time.Date(2025, 4, 21, 8, 0, 0, 0, time.UTC),
				},
				limit: 2,
			},
			want:    []Module{},
			wantErr: false,
		},
	}

	CreateAndPopulateDatabase(t, "test_get_modules_db", moduleTestDataSQL)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAllModules(tt.args.orderBy, tt.args.filter, tt.args.lastModule, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetAllModules() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want != nil {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("GetAllModules() mismatch (-want +got):\n%s", diff)
					t.Errorf("want:\n%s", formatModules(tt.want))
					t.Errorf(" got:\n%s", formatModules(got))
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
