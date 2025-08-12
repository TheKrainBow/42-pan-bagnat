package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	apiManager "github.com/TheKrainBow/go-api"
)

// ========== Root user ==========
// Holds all the scalar properties that appear both at the root and inside cursus_users[].user
type UserCore42 struct {
	ID              int        `json:"id"`
	Email           string     `json:"email"`
	Login           string     `json:"login"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	UsualFullName   string     `json:"usual_full_name"`
	UsualFirstName  *string    `json:"usual_first_name"`
	URL             string     `json:"url"`
	Phone           *string    `json:"phone"`
	Displayname     string     `json:"displayname"`
	Kind            string     `json:"kind"`
	Image           Image42    `json:"image"`
	Staff           bool       `json:"staff?"`
	CorrectionPoint int        `json:"correction_point"`
	PoolMonth       *string    `json:"pool_month"`
	PoolYear        *string    `json:"pool_year"`
	Location        *string    `json:"location"`
	Wallet          int        `json:"wallet"`
	AnonymizeDate   *time.Time `json:"anonymize_date"`
	DataErasureDate *time.Time `json:"data_erasure_date"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	AlumnizedAt     *time.Time `json:"alumnized_at"`
	Alumni          bool       `json:"alumni?"`
	Active          bool       `json:"active?"`
}

// Root user embeds core + collections
type User42 struct {
	UserCore42

	Groups          []Group42         `json:"groups"`
	CursusUsers     []CursusUser42    `json:"cursus_users"`
	ProjectsUsers   []ProjectUser42   `json:"projects_users"`
	LanguagesUsers  []LanguageUser42  `json:"languages_users"`
	Achievements    []Achievement42   `json:"achievements"`
	Titles          []Title42         `json:"titles"`
	TitlesUsers     []TitleUser42     `json:"titles_users"`
	Partnerships    []json.RawMessage `json:"partnerships" swaggertype:"array,object"`
	Patroned        []Patroned42      `json:"patroned"`
	Patroning       []json.RawMessage `json:"patroning"  swaggertype:"array,object"`
	ExpertisesUsers []ExpertiseUser42 `json:"expertises_users"`
	Roles           []Role42          `json:"roles"`
	Campus          []Campus42        `json:"campus"`
	CampusUsers     []CampusUser42    `json:"campus_users"`
}

// Nested user just needs the scalar bits
type UserRef42 struct {
	UserCore42
}

// ========== Simple nested types ==========

type Group42 struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Title42 struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TitleUser42 struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	TitleID   int        `json:"title_id"`
	Selected  bool       `json:"selected"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type Role42 struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Image42 struct {
	Link     string          `json:"link"`
	Versions ImageVersions42 `json:"versions"`
}

type ImageVersions42 struct {
	Large  string `json:"large"`
	Medium string `json:"medium"`
	Small  string `json:"small"`
	Micro  string `json:"micro"`
}

// ========== Cursus ==========

type CursusUser42 struct {
	ID           int        `json:"id"`
	BeginAt      *time.Time `json:"begin_at"`
	EndAt        *time.Time `json:"end_at"`
	Grade        *string    `json:"grade"`
	Level        float64    `json:"level"`
	Skills       []Skill42  `json:"skills"`
	CursusID     int        `json:"cursus_id"`
	HasCoalition bool       `json:"has_coalition"`
	BlackholedAt *time.Time `json:"blackholed_at"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	Cursus       Cursus42   `json:"cursus"`
}

type Skill42 struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Level float64 `json:"level"`
}

type Cursus42 struct {
	ID        int        `json:"id"`
	CreatedAt *time.Time `json:"created_at"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Kind      string     `json:"kind"`
}

// ========== Projects ==========

type ProjectUser42 struct {
	ID            int        `json:"id"`
	Occurrence    int        `json:"occurrence"`
	FinalMark     *int       `json:"final_mark"`
	Status        string     `json:"status"`     // e.g., in_progress, finished, searching_a_group
	Validated     *bool      `json:"validated?"` // bool or null
	CurrentTeamID *int       `json:"current_team_id"`
	Project       Project42  `json:"project"`
	CursusIDs     []int      `json:"cursus_ids"`
	MarkedAt      *time.Time `json:"marked_at"`
	Marked        bool       `json:"marked"`
	RetriableAt   *time.Time `json:"retriable_at"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

type Project42 struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ParentID *int   `json:"parent_id"`
}

// ========== Languages ==========

type LanguageUser42 struct {
	ID         int        `json:"id"`
	LanguageID int        `json:"language_id"`
	UserID     int        `json:"user_id"`
	Position   int        `json:"position"`
	CreatedAt  *time.Time `json:"created_at"`
}

// ========== Achievements ==========

type Achievement42 struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Tier         string `json:"tier"`
	Kind         string `json:"kind"`
	Visible      bool   `json:"visible"`
	Image        string `json:"image"`
	NbrOfSuccess *int   `json:"nbr_of_success"`
	UsersURL     string `json:"users_url"`
}

// ========== Patronage / Expertise (from earlier sample) ==========

type Patroned42 struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	GodfatherID int        `json:"godfather_id"`
	Ongoing     bool       `json:"ongoing"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type Patroning42 struct {
	// Schema not shown in samples—keep a placeholder for symmetry.
	// Add fields when you have a concrete payload.
	// Using RawMessage would also be acceptable if you don't use it in rules.
}

type ExpertiseUser42 struct {
	ID          int        `json:"id"`
	ExpertiseID int        `json:"expertise_id"`
	Interested  bool       `json:"interested"`
	Value       int        `json:"value"`
	ContactMe   bool       `json:"contact_me"`
	CreatedAt   *time.Time `json:"created_at"`
	UserID      int        `json:"user_id"`
}

// ========== Campus ==========

type Campus42 struct {
	ID                 int          `json:"id"`
	Name               string       `json:"name"`
	TimeZone           string       `json:"time_zone"`
	Language           CampusLang42 `json:"language"`
	UsersCount         int          `json:"users_count"`
	VogsphereID        int          `json:"vogsphere_id"`
	Country            string       `json:"country"`
	Address            string       `json:"address"`
	ZIP                string       `json:"zip"`
	City               string       `json:"city"`
	Website            string       `json:"website"`
	Facebook           string       `json:"facebook"`
	Twitter            string       `json:"twitter"`
	Active             bool         `json:"active"`
	Public             bool         `json:"public"`
	EmailExtension     string       `json:"email_extension"`
	DefaultHiddenPhone bool         `json:"default_hidden_phone"`
}

type CampusLang42 struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Identifier string     `json:"identifier"`
	CreatedAt  *time.Time `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

type CampusUser42 struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	CampusID  int        `json:"campus_id"`
	IsPrimary bool       `json:"is_primary"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func GetUser42(login string) (User42, error) {
	var out User42

	path := fmt.Sprintf("/users/%s", url.PathEscape(login))
	resp, err := apiManager.GetClient("42").Get(path)
	if err != nil {
		return out, fmt.Errorf("42 API GET %s: %w", path, err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		var apiErr struct {
			Error            string `json:"error"`
			Message          string `json:"message"`
			ErrorDescription string `json:"error_description"`
		}
		msg := strings.TrimSpace(string(body))
		if json.Unmarshal(body, &apiErr) == nil {
			switch {
			case apiErr.Message != "":
				msg = apiErr.Message
			case apiErr.ErrorDescription != "":
				msg = apiErr.ErrorDescription
			case apiErr.Error != "":
				msg = apiErr.Error
			}
		}
		return out, fmt.Errorf("42 API GET %s: status %d: %s", path, resp.StatusCode, msg)
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return out, fmt.Errorf("42 API decode %s: %w", path, err)
	}

	return out, nil
}
