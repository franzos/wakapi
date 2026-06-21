package models

import (
	"time"

	"github.com/duke-git/lancet/v2/slice"
)

const (
	ShareChartActivity   = "activity"
	ShareChartLanguages  = "languages"
	ShareChartEditors    = "editors"
	ShareChartOS         = "operating_systems"
	ShareChartCategories = "categories"
)

var ShareChartTypes = []string{
	ShareChartActivity,
	ShareChartLanguages,
	ShareChartEditors,
	ShareChartOS,
	ShareChartCategories,
}

// ShareIntervals is the allow-list of range aliases (a subset of models.AllIntervals).
var ShareIntervals = []string{"last_7_days", "last_30_days", "last_year", "all_time"}

var ShareThemes = []string{"light", "dark"}

type Share struct {
	ID        string    `json:"id" gorm:"primary_key; size:36"`
	User      *User     `json:"-" gorm:"not null; constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UserID    string    `json:"-" gorm:"not null; index:idx_share_user"`
	ChartType string    `json:"chart_type" gorm:"type:varchar(32); not null"`
	Interval  string    `json:"interval" gorm:"type:varchar(32); not null"`
	Theme     string    `json:"theme" gorm:"type:varchar(16); not null; default:light"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (s *Share) IsValid() bool {
	return s.ID != "" &&
		s.UserID != "" &&
		slice.Contain(ShareChartTypes, s.ChartType) &&
		slice.Contain(ShareIntervals, s.Interval) &&
		slice.Contain(ShareThemes, s.Theme)
}
