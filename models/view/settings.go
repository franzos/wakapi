package view

import (
	"time"

	"github.com/muety/wakapi/models"
)

type SettingsViewModel struct {
	SharedLoggedInViewModel
	LanguageMappings      []*models.LanguageMapping
	Aliases               []*SettingsVMCombinedAlias
	Labels                []*SettingsVMCombinedLabel
	Projects              []string
	SubscriptionPrice     string
	DataRetentionMonths   int
	UserFirstData         time.Time
	SupportContact        string
	InviteLink            string
	ReadmeCardCustomTitle string
	ApiKeys               []*SettingsApiKeys
	WebAuthnCredentials   []*models.WebAuthnCredential
	DisableWebAuthn       bool
	Shares                []*SettingsShare
	SharesEnabled         bool
}

type SettingsVMCombinedAlias struct {
	Key    string
	Type   uint8
	Values []string
}

type SettingsVMCombinedLabel struct {
	Key    string
	Values []string
}

type SettingsApiKeys struct {
	Name     string
	Value    string
	ReadOnly bool
}

type SettingsShare struct {
	ID        string
	Title     string
	ChartType string
	Interval  string
	Theme     string
	SvgUrl    string
	JsonUrl   string
	Embed     string
}

func (s *SettingsViewModel) SubscriptionsEnabled() bool {
	return s.SubscriptionPrice != ""
}

func (s *SettingsViewModel) WithSuccess(m string) *SettingsViewModel {
	s.SetSuccess(m)
	return s
}

func (s *SettingsViewModel) WithError(m string) *SettingsViewModel {
	s.SetError(m)
	return s
}
