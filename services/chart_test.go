package services

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
)

func TestChartService_RenderBreakdown(t *testing.T) {
	config.Set(config.Empty())

	user := &models.User{ID: "user1"}
	summary := &models.Summary{
		User:     user,
		UserID:   "user1",
		FromTime: models.CustomTime(time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)),
		ToTime:   models.CustomTime(time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)),
		Languages: []*models.SummaryItem{
			{Type: models.SummaryLanguage, Key: "Go", Total: 90 * time.Minute / time.Second},
			{Type: models.SummaryLanguage, Key: "Rust", Total: 30 * time.Minute / time.Second},
		},
	}

	summaryMock := new(mocks.SummaryServiceMock)
	summaryMock.On("Aliased", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), user, mock.AnythingOfType("types.SummaryRetriever"), mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	sut := NewChartService(summaryMock)

	svg, err := sut.RenderChart(user, models.ShareChartLanguages, models.IntervalPast7Days, false, true)

	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(svg, "<svg"))
	assert.Contains(t, svg, "Languages")
	assert.Contains(t, svg, "Go")
	assert.Contains(t, svg, "75%") // 90 of 120 mins
}

func TestChartService_RenderActivity(t *testing.T) {
	config.Set(config.Empty())

	user := &models.User{ID: "user1"}
	daySummary := &models.Summary{
		User: user, UserID: "user1",
		FromTime: models.CustomTime(time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)),
		ToTime:   models.CustomTime(time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)),
		Projects: []*models.SummaryItem{{Type: models.SummaryProject, Key: "wakapi", Total: 60 * time.Minute / time.Second}},
	}

	summaryMock := new(mocks.SummaryServiceMock)
	summaryMock.On("Retrieve", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), user, mock.Anything, mock.Anything).Return(daySummary, nil)

	sut := NewChartService(summaryMock)

	svg, err := sut.RenderChart(user, models.ShareChartActivity, models.IntervalPast7Days, true, true)

	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(svg, "<svg"))
	assert.Contains(t, svg, "Coding Activity")
}

func TestChartService_ActivitySeries(t *testing.T) {
	config.Set(config.Empty())

	user := &models.User{ID: "user1"}
	daySummary := &models.Summary{
		User: user, UserID: "user1",
		FromTime: models.CustomTime(time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)),
		ToTime:   models.CustomTime(time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)),
		Projects: []*models.SummaryItem{{Type: models.SummaryProject, Key: "wakapi", Total: 60 * time.Minute / time.Second}},
	}

	summaryMock := new(mocks.SummaryServiceMock)
	summaryMock.On("Retrieve", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), user, mock.Anything, mock.Anything).Return(daySummary, nil)

	sut := NewChartService(summaryMock)

	from := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)
	points, err := sut.ActivitySeries(user, from, to)

	assert.NoError(t, err)
	assert.NotEmpty(t, points)
	var total time.Duration
	for _, p := range points {
		total += p.Total
	}
	assert.Positive(t, total)
}
