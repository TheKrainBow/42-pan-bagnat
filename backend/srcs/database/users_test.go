package database

import (
	_ "embed"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

//go:embed test_data/test_data_users.sql
var testDataSQL string

func TestGetAllUsers(t *testing.T) {
	type args struct {
		orderBy  *[]UserOrder
		filter   string
		lastUser *User
		limit    int
	}
	tests := []struct {
		name    string
		args    args
		want    []User
		wantIDs []string
		wantErr bool
	}{
		// STRUCTURE TEST
		{
			name: "STRUCTURE TEST: All datas",
			args: args{
				orderBy:  nil,
				lastUser: nil,
				limit:    -8,
			},
			want: []User{
				{ID: "user_01HZXYZDE0420", FtLogin: "heinz", FtID: 220393, FtIsStaff: true, PhotoURL: "https://intra.42.fr/heinz/220393", LastSeen: time.Date(2001, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "user_01HZXYZDE0430", FtLogin: "ltcherep", FtID: 194037, FtIsStaff: false, PhotoURL: "https://intra.42.fr/ltcherep/194037", LastSeen: time.Date(2000, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "user_01HZXYZDE0440", FtLogin: "tac", FtID: 79125, FtIsStaff: true, PhotoURL: "https://intra.42.fr/tac/79125", LastSeen: time.Date(2003, 4, 16, 12, 0, 0, 0, time.UTC)},
				{ID: "user_01HZXYZDE0450", FtLogin: "yoshi", FtID: 78574, FtIsStaff: true, PhotoURL: "https://intra.42.fr/yoshi/78574", LastSeen: time.Date(2002, 4, 16, 12, 0, 0, 0, time.UTC)},
			},
			wantErr: false,
		},
		// LIMIT TESTS:
		{
			name: "LIMIT TESTS: Negative",
			args: args{
				orderBy:  nil,
				lastUser: nil,
				limit:    -8,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0450",
			},
			wantErr: false,
		},
		{
			name: "LIMIT TESTS: 0",
			args: args{
				orderBy:  nil,
				lastUser: nil,
				limit:    0,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0450",
			},
			wantErr: false,
		},
		{
			name: "LIMIT TESTS: 1",
			args: args{
				orderBy:  nil,
				lastUser: nil,
				limit:    1,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
			},
			wantErr: false,
		},
		{
			name: "LIMIT TESTS: 2",
			args: args{
				orderBy:  nil,
				lastUser: nil,
				limit:    2,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
			},
			wantErr: false,
		},
		{
			name: "LIMIT TESTS: 3",
			args: args{
				orderBy:  nil,
				lastUser: nil,
				limit:    3,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0440",
			},
			wantErr: false,
		},
		{
			name: "LIMIT TESTS: 4",
			args: args{
				orderBy:  nil,
				lastUser: nil,
				limit:    4,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0450",
			},
			wantErr: false,
		},
		// ORDER TEST:
		{
			name: "ORDER TEST: Asc on FtLogin",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserFtLogin, Order: Asc},
				},
				lastUser: nil,
				limit:    0,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0450",
			},
			wantErr: false,
		},
		{
			name: "ORDER TEST: Desc on FtLogin",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserFtLogin, Order: Desc},
				},
				lastUser: nil,
				limit:    0,
			},
			wantIDs: []string{
				"user_01HZXYZDE0450",
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0420",
			},
			wantErr: false,
		},
		{
			name: "ORDER TEST: Asc on LastSeen",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserLastSeen, Order: Asc},
				},
				lastUser: nil,
				limit:    0,
			},
			wantIDs: []string{
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0450",
				"user_01HZXYZDE0440",
			},
			wantErr: false,
		},
		{
			name: "ORDER TEST: Desc on LastSeen",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserLastSeen, Order: Desc},
				},
				lastUser: nil,
				limit:    0,
			},
			wantIDs: []string{
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0450",
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
			},
			wantErr: false,
		},
		// Pagination tests
		{
			name: "PAGINATION: Page size 2, Page number 2",
			args: args{
				orderBy:  nil,
				lastUser: &User{ID: "user_01HZXYZDE0430", FtLogin: "ltcherep", FtID: 194037, FtIsStaff: false, PhotoURL: "https://intra.42.fr/ltcherep/194037", LastSeen: time.Date(2000, 04, 16, 12, 0, 0, 0, time.UTC)},
				limit:    2,
			},
			wantIDs: []string{
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0450",
			},
			wantErr: false,
		},
		{
			name: "PAGINATION: Page size 2, Page number 1, Ordered",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserLastSeen, Order: Desc},
				},
				lastUser: nil,
				limit:    2,
			},
			wantIDs: []string{
				"user_01HZXYZDE0440",
				"user_01HZXYZDE0450",
			},
			wantErr: false,
		},
		{
			name: "PAGINATION: Page size 2, Page number 2, Ordered",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserLastSeen, Order: Desc},
				},
				lastUser: &User{ID: "user_01HZXYZDE0450", FtLogin: "yoshi", FtID: 78574, FtIsStaff: true, PhotoURL: "https://intra.42.fr/yoshi/78574", LastSeen: time.Date(2002, 04, 16, 12, 0, 0, 0, time.UTC)},
				limit:    2,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
				"user_01HZXYZDE0430",
			},
			wantErr: false,
		},
		// Filter Test
		{
			name: "FILTER TEST: heinz",
			args: args{
				orderBy:  nil,
				filter:   "heinz",
				lastUser: nil,
				limit:    4,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
			},
			wantErr: false,
		},
		{
			name: "FILTER TEST: t",
			args: args{
				orderBy:  nil,
				filter:   "t",
				lastUser: nil,
				limit:    4,
			},
			wantIDs: []string{
				"user_01HZXYZDE0430",
				"user_01HZXYZDE0440",
			},
			wantErr: false,
		},
		{
			name: "FILTER TEST: t, pagination 1 page 1",
			args: args{
				orderBy:  nil,
				filter:   "t",
				lastUser: nil,
				limit:    1,
			},
			wantIDs: []string{
				"user_01HZXYZDE0430",
			},
			wantErr: false,
		},
		{
			name: "FILTER TEST: t, pagination 2 page 2",
			args: args{
				orderBy:  nil,
				filter:   "t",
				lastUser: &User{ID: "user_01HZXYZDE0430", FtLogin: "ltcherep", FtID: 194037, FtIsStaff: false, PhotoURL: "https://intra.42.fr/ltcherep/194037", LastSeen: time.Date(2000, 04, 16, 12, 0, 0, 0, time.UTC)},
				limit:    1,
			},
			wantIDs: []string{
				"user_01HZXYZDE0440",
			},
			wantErr: false,
		},
		{
			name: "FILTER TEST: Filter \"2\", pagination 1 page 1, order FtLogin Asc",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserFtLogin, Order: Asc},
				},
				filter:   "2",
				lastUser: nil,
				limit:    1,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
			},
			wantErr: false,
		},
		{
			name: "FILTER TEST: Filter \"2\", pagination 1 page 2, order FtLogin Asc",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserFtLogin, Order: Asc},
				},
				filter:   "2",
				lastUser: &User{ID: "user_01HZXYZDE0420", FtLogin: "heinz", FtID: 220393, FtIsStaff: true, PhotoURL: "https://intra.42.fr/heinz/220393", LastSeen: time.Date(2001, 04, 16, 12, 0, 0, 0, time.UTC)},
				limit:    1,
			},
			wantIDs: []string{
				"user_01HZXYZDE0440",
			},
			wantErr: false,
		},
		{
			name: "FILTER TEST: Filter \"2\", pagination 1 page 1, order FtLogin Desc",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserFtLogin, Order: Desc},
				},
				filter:   "2",
				lastUser: nil,
				limit:    1,
			},
			wantIDs: []string{
				"user_01HZXYZDE0440",
			},
			wantErr: false,
		},
		{
			name: "FILTER TEST: Filter \"2\", pagination 1 page 2, order FtLogin Desc",
			args: args{
				orderBy: &[]UserOrder{
					{Field: UserFtLogin, Order: Desc},
				},
				filter:   "2",
				lastUser: &User{ID: "user_01HZXYZDE0440", FtLogin: "tac", FtID: 79125, FtIsStaff: true, PhotoURL: "https://intra.42.fr/tac/79125", LastSeen: time.Date(2003, 04, 16, 12, 0, 0, 0, time.UTC)},
				limit:    1,
			},
			wantIDs: []string{
				"user_01HZXYZDE0420",
			},
			wantErr: false,
		},
	}
	CreateAndPopulateDatabase(t, "test_get_user_db", testDataSQL)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAllUsers(tt.args.orderBy, tt.args.filter, tt.args.lastUser, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("GetAllUsers() mismatch (-want +got):\n%s", diff)
					t.Errorf("want:\n%s", formatUsers(tt.want))
					t.Errorf(" got:\n%s", formatUsers(got))
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
