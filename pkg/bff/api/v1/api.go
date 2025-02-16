package api

import (
	"context"

	"github.com/trisacrypto/directory/pkg/bff/db/models/v1"
	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
)

//===========================================================================
// Service Interface
//===========================================================================

type BFFClient interface {
	// Unauthenticated Endpoints
	Status(context.Context, *StatusParams) (*StatusReply, error)
	Lookup(context.Context, *LookupParams) (*LookupReply, error)
	VerifyContact(context.Context, *VerifyContactParams) (*VerifyContactReply, error)

	// User Management Endpoints
	Login(context.Context) error

	// Authenticated Endpoints
	LoadRegistrationForm(context.Context) (*models.RegistrationForm, error)
	SaveRegistrationForm(context.Context, *models.RegistrationForm) error
	SubmitRegistration(_ context.Context, network string) (*RegisterReply, error)
	RegistrationStatus(context.Context) (*RegistrationStatus, error)
	Overview(context.Context) (*OverviewReply, error)
	Announcements(context.Context) (*AnnouncementsReply, error)
	MakeAnnouncement(context.Context, *models.Announcement) error
	Certificates(context.Context) (*CertificatesReply, error)
	MemberDetails(context.Context, *MemberDetailsParams) (*MemberDetailsReply, error)
	Attention(context.Context) (*AttentionReply, error)
}

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors
type Reply struct {
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty" yaml:"error,omitempty"`
	RefreshToken bool   `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
}

// StatusParams is parsed from the query parameters of the GET request
type StatusParams struct {
	NoGDS bool `url:"nogds,omitempty" form:"nogds" default:"false"`
}

// StatusReply is returned on status requests. Note that no request is needed.
type StatusReply struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
	TestNet string `json:"testnet,omitempty"`
	MainNet string `json:"mainnet,omitempty"`
}

//===========================================================================
// BFF v1 API Requests and Responses
//===========================================================================

// LookupParams is converted into a GDS LookupRequest.
type LookupParams struct {
	ID         string `url:"uuid,omitempty" form:"uuid"`
	CommonName string `url:"common_name,omitempty" form:"common_name"`
}

// LookupReply can return 1-2 results either one result found from one directory
// service or results found from both TestNet and MainNet. If no results are found, the
// Lookup endpoint returns a 404 error (not found). The result is the simplest case,
// just a JSON serialization of the protocol buffers returned from GDS to help long term
// maintainability. The protocol buffers contain a "registered_directory" field that
// will have either vaspdirectory.net or trisatest.net inside of it - which can be used
// to identify which network the record is associated with. The protocol buffers may
// also contain an "error" field - the BFF will handle this field by logging the error
// but will exclude it from any results returned.
type LookupReply struct {
	TestNet map[string]interface{} `json:"testnet"`
	MainNet map[string]interface{} `json:"mainnet"`
}

// VerifyContactParams is converted into a GDS VerifyContactRequest.
type VerifyContactParams struct {
	ID        string `url:"vaspID,omitempty" form:"vaspID"`
	Token     string `url:"token,omitempty" form:"token"`
	Directory string `url:"registered_directory,omitempty" form:"registered_directory"`
}

// VerifyContactReply
type VerifyContactReply struct {
	Error   map[string]interface{} `json:"error,omitempty"`
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
}

// RegisterReply is converted from a protocol buffer RegisterReply.
type RegisterReply struct {
	Error               map[string]interface{} `json:"error,omitempty"`
	Id                  string                 `json:"id"`
	RegisteredDirectory string                 `json:"registered_directory"`
	CommonName          string                 `json:"common_name"`
	Status              string                 `json:"status"`
	Message             string                 `json:"message"`
	PKCS12Password      string                 `json:"pkcs12password"`
}

// RegistrationStatus is returned on registration status requests. This will contain
// RFC3339 formatted timestamps indicating when the registration was submitted for
// testnet and mainnet.
type RegistrationStatus struct {
	TestNetSubmitted string `json:"testnet_submitted,omitempty"`
	MainNetSubmitted string `json:"mainnet_submitted,omitempty"`
}

// OverviewReply is returned on overview requests.
type OverviewReply struct {
	Error   NetworkError    `json:"error,omitempty"`
	OrgID   string          `json:"org_id"`
	TestNet NetworkOverview `json:"testnet"`
	MainNet NetworkOverview `json:"mainnet"`
}

// NetworkOverview contains network-specific information.
type NetworkOverview struct {
	Status             string        `json:"status"`
	Vasps              int           `json:"vasps"`
	CertificatesIssued int           `json:"certificates_issued"`
	NewMembers         int           `json:"new_members"`
	MemberDetails      MemberDetails `json:"member_details"`
}

// MemberDetails contains VASP-specific information.
type MemberDetails struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"`
	CountryCode string                 `json:"country_code"`
	Certificate map[string]interface{} `json:"certificate"`
}

// AnnouncementsReply contains up to the last 10 network announcements that were made in
// the past month. It does not require pagination since only relevant results are returned.
type AnnouncementsReply struct {
	Announcements []*models.Announcement `json:"announcements"`
	LastUpdated   string                 `json:"last_updated,omitempty"`
}

// CertificatesReply is returned on certificates requests.
type CertificatesReply struct {
	Error   NetworkError  `json:"network_error,omitempty"`
	TestNet []Certificate `json:"testnet"`
	MainNet []Certificate `json:"mainnet"`
}

// Certificate contains details about a certificate issued to a VASP.
type Certificate struct {
	SerialNumber string                 `json:"serial_number"`
	IssuedAt     string                 `json:"issued_at"`
	ExpiresAt    string                 `json:"expires_at"`
	Revoked      bool                   `json:"revoked"`
	Details      map[string]interface{} `json:"details"`
}

// MemberDetailsParams contains details required to identify a VASP member for the
// MembersDetails request.
type MemberDetailsParams struct {
	ID        string `url:"vaspID,omitempty" form:"vaspID"`
	Directory string `url:"registered_directory,omitempty" form:"registered_directory"`
}

// MemberDetailsReply contains sensitive details about a VASP member.
type MemberDetailsReply struct {
	Summary     *members.VASPMember    `json:"summary"`
	LegalPerson map[string]interface{} `json:"legal_person"`
	Trixo       map[string]interface{} `json:"trixo"`
}

// AttentionReply contains all the current attention messages relevant to an
// organization.
type AttentionReply struct {
	Messages []*AttentionMessage `json:"messages"`
}

// AttentionMessage contains details about a single attention message.
type AttentionMessage struct {
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Action   string `json:"action"`
}

// NetworkError is populated when the BFF receives an error from a network endpoint,
// containing an error string for each network that errored. This allows the client to
// distinguish between network errors and BFF errors and determine which network the
// errors originated from.
type NetworkError struct {
	TestNet string `json:"testnet,omitempty"`
	MainNet string `json:"mainnet,omitempty"`
}
