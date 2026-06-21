package routes

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	v1 "github.com/muety/wakapi/models/compat/wakatime/v1"
	"github.com/muety/wakapi/services"
)

var (
	shareExtPattern = regexp.MustCompile(`\.(svg|json)$`)
	jsonpCallback   = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$.]*$`)
)

type ShareHandler struct {
	config         *conf.Config
	userService    services.IUserService
	shareService   services.IShareService
	summaryService services.ISummaryService
	chartService   services.IChartService
}

func NewShareHandler(userService services.IUserService, shareService services.IShareService, summaryService services.ISummaryService, chartService services.IChartService) *ShareHandler {
	return &ShareHandler{
		config:         conf.Get(),
		userService:    userService,
		shareService:   shareService,
		summaryService: summaryService,
		chartService:   chartService,
	}
}

func (h *ShareHandler) RegisterRoutes(router chi.Router) {
	r := chi.NewRouter()
	r.Get("/{user}/{idWithExt}", h.Get)
	router.Mount("/share", r)
}

func (h *ShareHandler) Get(w http.ResponseWriter, r *http.Request) {
	// chi currently doesn't support dots in parameters of routes containing a dot themselves, this is a workaround
	// https://github.com/go-chi/chi/issues/758
	userParam := strings.TrimPrefix(chi.URLParam(r, "user"), "@")
	idWithExt := chi.URLParam(r, "idWithExt")

	loc := shareExtPattern.FindStringSubmatchIndex(idWithExt)
	if loc == nil {
		h.notFound(w)
		return
	}
	ext := idWithExt[loc[2]:loc[3]]
	id := idWithExt[:loc[0]]

	share, err := h.shareService.GetById(id)
	if err != nil {
		h.notFound(w)
		return
	}

	user, err := h.userService.GetUserById(userParam)
	if err != nil || user.ID != share.UserID {
		h.notFound(w)
		return
	}
	share.User = user

	interval, err := helpers.ParseInterval(share.Interval)
	if err != nil {
		h.notFound(w)
		return
	}

	switch ext {
	case "svg":
		h.respondSvg(w, r, user, share, interval)
	case "json":
		h.respondJson(w, r, user, share, interval)
	default:
		h.notFound(w)
	}
}

func (h *ShareHandler) respondSvg(w http.ResponseWriter, r *http.Request, user *models.User, share *models.Share, interval *models.IntervalKey) {
	dark := share.Theme == "dark"
	if q := r.URL.Query().Get("theme"); q == "dark" {
		dark = true
	} else if q == "light" {
		dark = false
	}

	chart, err := h.chartService.RenderChart(user, share.ChartType, interval, dark, false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		conf.Log().Request(r).Error("failed to render share chart", "shareID", share.ID, "error", err)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "max-age=21600") // 6 hours
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(chart))
}

type shareActivityResponse struct {
	Data shareActivityData `json:"data"`
}

type shareActivityData struct {
	ChartType          string                `json:"chart_type"`
	Range              string                `json:"range"`
	Start              string                `json:"start"`
	End                string                `json:"end"`
	TotalSeconds       float64               `json:"total_seconds"`
	HumanReadableTotal string                `json:"human_readable_total"`
	Buckets            []shareActivityBucket `json:"buckets"`
}

type shareActivityBucket struct {
	Label        string  `json:"label"`
	TotalSeconds float64 `json:"total_seconds"`
	Text         string  `json:"text"`
}

func (h *ShareHandler) respondJson(w http.ResponseWriter, r *http.Request, user *models.User, share *models.Share, interval *models.IntervalKey) {
	err, from, to := helpers.ResolveIntervalTZ(interval, user.TZ(), user.StartOfWeekDay())
	if err != nil {
		h.notFound(w)
		return
	}

	if share.ChartType == models.ShareChartActivity {
		points, err := h.chartService.ActivitySeries(user, from, to)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			conf.Log().Request(r).Error("failed to load activity series for share", "shareID", share.ID, "error", err)
			return
		}

		var total time.Duration
		buckets := make([]shareActivityBucket, len(points))
		for i, p := range points {
			total += p.Total
			buckets[i] = shareActivityBucket{
				Label:        p.Label,
				TotalSeconds: p.Total.Seconds(),
				Text:         helpers.FmtWakatimeDuration(p.Total),
			}
		}

		h.writeShareJSON(w, r, shareActivityResponse{Data: shareActivityData{
			ChartType:          share.ChartType,
			Range:              share.Interval,
			Start:              from.Format(time.RFC3339),
			End:                to.Format(time.RFC3339),
			TotalSeconds:       total.Seconds(),
			HumanReadableTotal: helpers.FmtWakatimeDuration(total),
			Buckets:            buckets,
		}})
		return
	}

	filters := &models.Filters{}
	summary, err := h.summaryService.Aliased(from, to, user, h.summaryService.Retrieve, filters, nil, false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		conf.Log().Request(r).Error("failed to load summary for share", "shareID", share.ID, "error", err)
		return
	}
	summary.User = user

	stats := v1.NewStatsFrom(summary, filters)
	stats.Data.Range = share.Interval
	stats.Data.HumanReadableRange = interval.GetHumanReadable()
	stats.Data.IsCodingActivityVisible = true
	stats.Data.IsOtherUsageVisible = true

	// expose only the breakdown the share renders; empty all others
	languages, editors, oss, categories := stats.Data.Languages, stats.Data.Editors, stats.Data.OperatingSystems, stats.Data.Categories
	empty := []*v1.SummariesEntry{}
	stats.Data.Editors = empty
	stats.Data.Languages = empty
	stats.Data.Machines = empty
	stats.Data.Projects = empty
	stats.Data.OperatingSystems = empty
	stats.Data.Branches = empty
	stats.Data.Categories = empty
	switch share.ChartType {
	case models.ShareChartLanguages:
		stats.Data.Languages = languages
	case models.ShareChartEditors:
		stats.Data.Editors = editors
	case models.ShareChartOS:
		stats.Data.OperatingSystems = oss
	case models.ShareChartCategories:
		stats.Data.Categories = categories
	}

	h.writeShareJSON(w, r, stats)
}

func (h *ShareHandler) writeShareJSON(w http.ResponseWriter, r *http.Request, obj any) {
	// JSONP fallback for cross-origin reads
	if cb := r.URL.Query().Get("callback"); cb != "" && jsonpCallback.MatchString(cb) {
		body, _ := json.Marshal(obj)
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "max-age=21600")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cb + "("))
		w.Write(body)
		w.Write([]byte(");"))
		return
	}
	w.Header().Set("Cache-Control", "max-age=21600")
	helpers.RespondJSON(w, r, http.StatusOK, obj)
}

func (h *ShareHandler) notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(conf.ErrNotFound))
}
