package users

import api "backend/api/dto"

// UserGetResponse is the paginated wrapper for a user list.
// swagger:model UserGetResponse
type UserGetResponse struct {
	// NextPageToken is the token to retrieve the next page of results.
	NextPageToken string `json:"next_page_token,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"`
	// Users is the list of users on this page.
	Users []api.User `json:"users"`
}

// UserPatchInput defines the fields you can modify on a user.
// swagger:model UserPatchInput
type UserPatchInput struct {
	// IsStaff toggles whether the user has staff privileges.
	IsStaff *bool `json:"is_staff,omitempty" example:"true"`
	// Roles is an optional list of role IDs to assign to the user.
	Roles []string `json:"roles,omitempty" example:"[\"role_01\",\"role_02\"]"`
}

// UserPostInput defines the payload for creating a new user.
// swagger:model UserPostInput
type UserPostInput struct {
	// FtID is the 42-intranet numeric ID of the user.
	FtID int `json:"ft_id"      example:"1492"`
	// FtLogin is the 42-intranet login handle.
	FtLogin string `json:"ft_login"   example:"heinz"`
	// FtPhoto is the URL to the user’s 42-intranet avatar.
	FtPhoto string `json:"ft_photo"   example:"https://intra.42.fr/some-login/some-id"`
	// IsStaff indicates whether the user has staff privileges in Pan Bagnat.
	IsStaff bool `json:"is_staff"   example:"false"`
	// Roles lists the role IDs to assign to this user upon creation.
	Roles []string `json:"roles,omitempty" example:"[\"role_01\",\"role_02\"]"`
}
