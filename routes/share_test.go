package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/services"
)

func TestShareHandler_Get(t *testing.T) {
	config.Set(config.Empty())

	user := &models.User{ID: "user1"}
	share := &models.Share{ID: "abc123", UserID: "user1", ChartType: models.ShareChartLanguages, Interval: "last_7_days", Theme: "light"}
	summary := &models.Summary{
		User: user, UserID: "user1",
		FromTime:  models.CustomTime(time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)),
		ToTime:    models.CustomTime(time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)),
		Languages: []*models.SummaryItem{{Type: models.SummaryLanguage, Key: "Go", Total: 60 * time.Minute / time.Second}},
		Editors:   []*models.SummaryItem{{Type: models.SummaryEditor, Key: "vim", Total: 60 * time.Minute / time.Second}},
		Projects:  []*models.SummaryItem{{Type: models.SummaryProject, Key: "wakapi", Total: 60 * time.Minute / time.Second}},
	}

	userMock := new(mocks.UserServiceMock)
	userMock.On("GetUserById", "user1").Return(user, nil)

	shareMock := new(mocks.ShareServiceMock)
	shareMock.On("GetById", "abc123").Return(share, nil)
	shareMock.On("GetById", "missing").Return(nil, assert.AnError)

	summaryMock := new(mocks.SummaryServiceMock)
	summaryMock.On("Aliased", mock.Anything, mock.Anything, user, mock.AnythingOfType("types.SummaryRetriever"), mock.Anything, mock.Anything, mock.Anything).Return(summary, nil)

	chartService := services.NewChartService(summaryMock)
	sut := NewShareHandler(userMock, shareMock, summaryMock, chartService)

	router := chi.NewRouter()
	sut.RegisterRoutes(router)

	serve := func(path string) *http.Response {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(rec, req)
		return rec.Result()
	}

	t.Run("renders svg for a valid share", func(t *testing.T) {
		res := serve("/share/@user1/abc123.svg")
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "image/svg+xml", res.Header.Get("Content-Type"))
		body, _ := io.ReadAll(res.Body)
		assert.True(t, strings.HasPrefix(string(body), "<svg"))
	})

	t.Run("returns wakatime json with only the shared breakdown", func(t *testing.T) {
		res := serve("/share/@user1/abc123.json")
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode)
		body, _ := io.ReadAll(res.Body)
		assert.Contains(t, string(body), `"languages"`)
		assert.Contains(t, string(body), `"data"`)
		assert.Contains(t, string(body), `"Go"`)
		// non-matching breakdowns must be emptied even though the summary has them
		assert.Contains(t, string(body), `"projects":[]`)
		assert.Contains(t, string(body), `"editors":[]`)
		assert.NotContains(t, string(body), `"vim"`)
		assert.NotContains(t, string(body), `"wakapi"`)
	})

	t.Run("returns focused activity json", func(t *testing.T) {
		activityShare := &models.Share{ID: "act456", UserID: "user1", ChartType: models.ShareChartActivity, Interval: "last_7_days", Theme: "light"}
		shareMock.On("GetById", "act456").Return(activityShare, nil)
		daySummary := &models.Summary{
			User: user, UserID: "user1",
			FromTime: models.CustomTime(time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)),
			ToTime:   models.CustomTime(time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)),
			Projects: []*models.SummaryItem{{Type: models.SummaryProject, Key: "wakapi", Total: 60 * time.Minute / time.Second}},
		}
		summaryMock.On("Retrieve", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), user, mock.Anything, mock.Anything).Return(daySummary, nil)

		res := serve("/share/@user1/act456.json")
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
		body, _ := io.ReadAll(res.Body)
		assert.Contains(t, string(body), `"chart_type":"activity"`)
		assert.Contains(t, string(body), `"buckets"`)
	})

	t.Run("404 for unknown share id", func(t *testing.T) {
		res := serve("/share/@user1/missing.svg")
		defer res.Body.Close()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("404 when username does not match share owner", func(t *testing.T) {
		userMock.On("GetUserById", "someoneelse").Return(&models.User{ID: "someoneelse"}, nil)
		res := serve("/share/@someoneelse/abc123.svg")
		defer res.Body.Close()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("404 for unsupported extension", func(t *testing.T) {
		res := serve("/share/@user1/abc123.png")
		defer res.Body.Close()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})
}
