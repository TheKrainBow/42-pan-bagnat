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
				{ID: "module_01HZXYZDE0420", Name: "Captain Hook", Slug: "captain-hook", Version: "1.2", Status: "enabled", GitURL: "https://github.com/42nice/captain-hook", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.7", LateCommits: 5, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0430", Name: "adm-stud", Slug: "adm-stud", Version: "1.5", Status: "enabled", GitURL: "https://github.com/42nice/adm-stud", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.5", LateCommits: 0, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0440", Name: "adm-manager", Slug: "adm-manager", Version: "1.0", Status: "enabled", GitURL: "https://github.com/42nice/adm-manager", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.0", LateCommits: 0, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0450", Name: "student-info", Slug: "student-info", Version: "1.8", Status: "enabled", GitURL: "https://github.com/42nice/student-info", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.9", LateCommits: 1, LastUpdate: time.Date(2025, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0460", Name: "role-manager", Slug: "role-manager", Version: "1.0", Status: "enabled", GitURL: "https://github.com/42nice/role-manager", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.0", LateCommits: 0, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0461", Name: "role-editor", Slug: "role-editor", Version: "1.1", Status: "enabled", GitURL: "https://github.com/42nice/role-editor", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.1", LateCommits: 2, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0462", Name: "support-tool", Slug: "support-tool", Version: "2.0", Status: "enabled", GitURL: "https://github.com/42nice/support-tool", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "2.0", LateCommits: 1, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0463", Name: "analytics", Slug: "analytics", Version: "3.2", Status: "enabled", GitURL: "https://github.com/42nice/analytics", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "3.2", LateCommits: 4, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0464", Name: "design-proto", Slug: "design-proto", Version: "0.9", Status: "enabled", GitURL: "https://github.com/42nice/design-proto", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "0.9", LateCommits: 0, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0465", Name: "test-suite", Slug: "test-suite", Version: "5.4", Status: "enabled", GitURL: "https://github.com/42nice/test-suite", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "5.4", LateCommits: 3, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0466", Name: "deploy-automate", Slug: "deploy-automate", Version: "1.5", Status: "enabled", GitURL: "https://github.com/42nice/deploy-automate", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.5", LateCommits: 0, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0467", Name: "strategy-dash", Slug: "strategy-dash", Version: "4.0", Status: "enabled", GitURL: "https://github.com/42nice/strategy-dash", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "4.0", LateCommits: 2, LastUpdate: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0468", Name: "zorbi-app", Slug: "zorbi-app", Version: "1.0", Status: "disabled", GitURL: "https://example.com/zorbi", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "1.0", LateCommits: 0, LastUpdate: time.Date(2025, 4, 21, 8, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0469", Name: "alpha-tool", Slug: "alpha-tool", Version: "2.2", Status: "disabled", GitURL: "https://example.com/alpha", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "2.2", LateCommits: 1, LastUpdate: time.Date(2025, 4, 21, 8, 0, 0, 0, time.UTC)},
				{ID: "module_01HZXYZDE0470", Name: "beta-service", Slug: "beta-service", Version: "3.5", Status: "enabled", GitURL: "https://example.com/beta", GitBranch: "main", IconURL: "https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg", LatestVersion: "3.5", LateCommits: 1, LastUpdate: time.Date(2025, 4, 21, 8, 0, 0, 0, time.UTC)},
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
					GitURL:        "https://github.com/42nice/adm-manager",
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
					GitURL:        "https://example.com/beta",
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

func TestPatchModule(t *testing.T) {
	CreateAndPopulateDatabase(t, "test_patch_module", moduleTestDataSQL)

	// Original module for comparison
	before, err := GetModule("module_01HZXYZDE0420")
	if err != nil {
		t.Fatalf("GetModule() before patch failed: %v", err)
	}

	name := "new-name"
	version := "9.9"
	lateCommits := 42
	patch := ModulePatch{
		ID:          before.ID,
		Name:        &name,
		Version:     &version,
		LateCommits: &lateCommits,
	}

	_, err = PatchModule(patch)
	if err != nil {
		t.Fatalf("PatchModule() failed: %v", err)
	}

	after, err := GetModule(before.ID)
	if err != nil {
		t.Fatalf("GetModule() after patch failed: %v", err)
	}

	if after.Name != name {
		t.Errorf("Name not patched: got %q, want %q", after.Name, name)
	}
	if after.Version != version {
		t.Errorf("Version not patched: got %q, want %q", after.Version, version)
	}
	if after.LateCommits != lateCommits {
		t.Errorf("LateCommits not patched: got %d, want %d", after.LateCommits, lateCommits)
	}
	if !after.LastUpdate.After(before.LastUpdate) {
		t.Errorf("LastUpdate not updated: before %v, after %v", before.LastUpdate, after.LastUpdate)
	}
}
