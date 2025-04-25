package core

import "testing"

func TestEncodeUserPaginationToken(t *testing.T) {
	type args struct {
		token UserPagination
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeUserPaginationToken(tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeUserPaginationToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EncodeUserPaginationToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
