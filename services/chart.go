package services

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	svg "github.com/ajstarks/svgo/float"
	"github.com/alitto/pond/v2"
	"github.com/patrickmn/go-cache"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
)

const (
	chartMaxBars = 8

	bdWidth   = 460.0
	bdBarLeft = 150.0
	bdBarW    = 220.0
	bdRowH    = 26.0
	bdPadTop  = 44.0
	bdPadX    = 16.0

	actWidth  = 460.0
	actBarMax = 120.0
	actBarW   = 14.0
	actBarGap = 6.0
	actPadTop = 44.0
	actPadX   = 16.0
	actBaseY  = 44.0 + 120.0 // padTop + barMax
)

// fallback palette for bars without a known color-map entry
var chartPalette = []string{"#2563eb", "#16a34a", "#dc2626", "#d97706", "#7c3aed", "#0891b2", "#db2777", "#65a30d"}

type chartTheme struct {
	bg, text, track, muted string
}

func themeFor(dark bool) chartTheme {
	if dark {
		return chartTheme{bg: "#1f2330", text: "#d1d5db", track: "#2b3040", muted: "#6b7280"}
	}
	return chartTheme{bg: "#ffffff", text: "#1f2937", track: "#e5e7eb", muted: "#9ca3af"}
}

type ChartService struct {
	config         *config.Config
	cache          *cache.Cache
	summaryService ISummaryService
}

func NewChartService(summaryService ISummaryService) *ChartService {
	return &ChartService{
		config:         config.Get(),
		cache:          cache.New(6*time.Hour, 6*time.Hour),
		summaryService: summaryService,
	}
}

// RenderChart returns an SVG document for the given share configuration.
func (s *ChartService) RenderChart(user *models.User, chartType string, interval *models.IntervalKey, darkTheme, skipCache bool) (string, error) {
	cacheKey := fmt.Sprintf("share_%s_%s_%s_%v", user.ID, chartType, (*interval)[0], darkTheme)
	if !skipCache {
		if v, found := s.cache.Get(cacheKey); found {
			return v.(string), nil
		}
	}

	err, from, to := helpers.ResolveIntervalTZ(interval, user.TZ(), user.StartOfWeekDay())
	if err != nil {
		return "", err
	}

	var out string
	if chartType == models.ShareChartActivity {
		out, err = s.renderActivity(user, from, to, darkTheme)
	} else {
		summaryType, ok := breakdownType(chartType)
		if !ok {
			return "", errors.New("unsupported chart type")
		}
		out, err = s.renderBreakdown(user, summaryType, from, to, darkTheme)
	}
	if err != nil {
		return "", err
	}

	s.cache.SetDefault(cacheKey, out)
	return out, nil
}

func breakdownType(chartType string) (uint8, bool) {
	switch chartType {
	case models.ShareChartLanguages:
		return models.SummaryLanguage, true
	case models.ShareChartEditors:
		return models.SummaryEditor, true
	case models.ShareChartOS:
		return models.SummaryOS, true
	case models.ShareChartCategories:
		return models.SummaryCategory, true
	}
	return 0, false
}

func breakdownTitle(summaryType uint8) string {
	switch summaryType {
	case models.SummaryLanguage:
		return "Languages"
	case models.SummaryEditor:
		return "Editors"
	case models.SummaryOS:
		return "Operating Systems"
	case models.SummaryCategory:
		return "Categories"
	}
	return ""
}

func (s *ChartService) renderBreakdown(user *models.User, summaryType uint8, from, to time.Time, dark bool) (string, error) {
	summary, err := s.summaryService.Aliased(from, to, user, s.summaryService.Retrieve, nil, nil, false)
	if err != nil {
		return "", err
	}

	items := *summary.GetByType(summaryType)
	sort.Sort(sort.Reverse(items))

	// percent of grand total, pre-truncation; TotalFixed keeps seconds-based units aligned
	var total time.Duration
	for _, it := range items {
		total += it.TotalFixed()
	}

	if len(items) > chartMaxBars {
		items = items[:chartMaxBars]
	}
	theme := themeFor(dark)

	rows := len(items)
	if rows == 0 {
		rows = 1
	}
	h := bdPadTop + float64(rows)*bdRowH + bdPadX

	buf := &bytes.Buffer{}
	canvas := svg.New(buf)
	canvas.Start(bdWidth, h)
	canvas.Rect(0, 0, bdWidth, h, fmt.Sprintf("fill:%s", theme.bg))
	canvas.Style("text/css",
		fmt.Sprintf("text{font-family:'Source Sans 3',Roboto,Helvetica,Arial,sans-serif;font-size:13px;fill:%s}", theme.text),
		"rect{rx:3px;ry:3px}",
	)
	canvas.Text(bdPadX, 26, breakdownTitle(summaryType), "font-size:16px;font-weight:600")

	if len(items) == 0 {
		canvas.Text(bdPadX, bdPadTop+18, "No data yet", fmt.Sprintf("fill:%s", theme.muted))
		canvas.End()
		return stripSVGHeader(buf.String()), nil
	}

	maxTotal := items[0].TotalFixed()
	for i, item := range items {
		y := bdPadTop + float64(i)*bdRowH
		dur := item.TotalFixed()
		frac := 0.0
		if maxTotal > 0 {
			frac = float64(dur) / float64(maxTotal)
		}
		pct := 0.0
		if total > 0 {
			pct = float64(dur) / float64(total) * 100
		}
		canvas.Text(bdPadX, y+15, truncateLabel(item.Key, 18))
		canvas.Rect(bdBarLeft, y+4, bdBarW, 14, fmt.Sprintf("fill:%s", theme.track))
		canvas.Rect(bdBarLeft, y+4, bdBarW*frac, 14, fmt.Sprintf("fill:%s", s.barColor(summaryType, item.Key, i)))
		canvas.Text(bdBarLeft+bdBarW+8, y+15, fmt.Sprintf("%.0f%%", pct))
	}

	canvas.End()
	return stripSVGHeader(buf.String()), nil
}

type ActivityPoint struct {
	Label string
	Total time.Duration
}

type activityRange struct {
	from, to time.Time
	label    string
}

// splitActivityRanges divides [from,to) into at most ~53 sub-ranges so the public
// activity endpoint can't fan out unboundedly (e.g. all_time = ~20k days).
func splitActivityRanges(from, to time.Time) []activityRange {
	spanDays := int(to.Sub(from).Hours()/24) + 1
	step := 1
	layout := "Jan 02"
	switch {
	case spanDays <= 31:
		step = 1
	case spanDays <= 371:
		step = 7
	default:
		step = (spanDays + 52) / 53 // ceil(spanDays/53)
		layout = "Jan 06"
	}
	var out []activityRange
	for cur := from; cur.Before(to); cur = cur.AddDate(0, 0, step) {
		end := cur.AddDate(0, 0, step)
		if end.After(to) {
			end = to
		}
		out = append(out, activityRange{from: cur, to: end, label: cur.Format(layout)})
	}
	return out
}

func (s *ChartService) ActivitySeries(user *models.User, from, to time.Time) ([]ActivityPoint, error) {
	ranges := splitActivityRanges(from, to)
	points := make([]ActivityPoint, len(ranges))

	wp := pond.NewPool(utils.HalfCPUs())
	mut := sync.Mutex{}
	for i, rg := range ranges {
		wp.Submit(func() {
			var total time.Duration
			sum, err := s.summaryService.Retrieve(rg.from, rg.to, user, nil, nil)
			if err != nil {
				config.Log().Warn("failed to retrieve summary for activity chart", "userID", user.ID, "error", err)
			} else {
				total = sum.TotalTime()
			}
			mut.Lock()
			points[i] = ActivityPoint{Label: rg.label, Total: total}
			mut.Unlock()
		})
	}
	wp.StopAndWait()

	return points, nil
}

func (s *ChartService) renderActivity(user *models.User, from, to time.Time, dark bool) (string, error) {
	buckets, err := s.ActivitySeries(user, from, to)
	if err != nil {
		return "", err
	}
	theme := themeFor(dark)

	w := actPadX*2 + float64(len(buckets))*(actBarW+actBarGap)
	if w < actWidth {
		w = actWidth
	}
	h := actPadTop + actBarMax + 28

	var maxTotal time.Duration
	for _, b := range buckets {
		if b.Total > maxTotal {
			maxTotal = b.Total
		}
	}

	buf := &bytes.Buffer{}
	canvas := svg.New(buf)
	canvas.Start(w, h)
	canvas.Rect(0, 0, w, h, fmt.Sprintf("fill:%s", theme.bg))
	canvas.Style("text/css",
		fmt.Sprintf("text{font-family:'Source Sans 3',Roboto,Helvetica,Arial,sans-serif;font-size:13px;fill:%s}", theme.text),
		"rect{rx:2px;ry:2px}",
	)
	canvas.Text(actPadX, 26, "Coding Activity", "font-size:16px;font-weight:600")

	for i, b := range buckets {
		x := actPadX + float64(i)*(actBarW+actBarGap)
		barH := 0.0
		if maxTotal > 0 {
			barH = actBarMax * float64(b.Total) / float64(maxTotal)
		}
		canvas.Group()
		canvas.Title(fmt.Sprintf("%s: %s", b.Label, helpers.FmtWakatimeDuration(b.Total)))
		canvas.Rect(x, actBaseY-barH, actBarW, barH, fmt.Sprintf("fill:%s", chartPalette[0]))
		canvas.Gend()
	}

	if len(buckets) > 0 {
		canvas.Text(actPadX, h-6, buckets[0].Label, fmt.Sprintf("font-size:11px;fill:%s", theme.muted))
		last := buckets[len(buckets)-1].Label
		canvas.Text(w-actPadX-float64(len(last))*6, h-6, last, fmt.Sprintf("font-size:11px;fill:%s", theme.muted))
	}

	canvas.End()
	return stripSVGHeader(buf.String()), nil
}

func (s *ChartService) barColor(summaryType uint8, key string, index int) string {
	var m map[string]string
	switch summaryType {
	case models.SummaryLanguage:
		m = s.config.App.GetLanguageColors()
	case models.SummaryEditor:
		m = s.config.App.GetEditorColors()
	case models.SummaryOS:
		m = s.config.App.GetOSColors()
	}
	if m != nil {
		if c, ok := m[strings.ToLower(key)]; ok && c != "" {
			if !strings.HasPrefix(c, "#") {
				c = "#" + c
			}
			return c
		}
	}
	return chartPalette[index%len(chartPalette)]
}

// drop svgo's XML decl + generator comment so output starts at <svg>
func stripSVGHeader(doc string) string {
	if i := strings.Index(doc, "<svg"); i > 0 {
		return doc[i:]
	}
	return doc
}

func truncateLabel(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
