package web

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/PedroJ03/asesor-inversiones/internal/config"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

// DefaultHistoryLimit is the number of recent alert snapshots used to compute
// the dashboard alert badge.
const DefaultHistoryLimit = 50

// DefaultAssetHistoryLimit is the default number of history rows shown on the
// asset detail page.
const DefaultAssetHistoryLimit = 100

// viewServer holds the read dependencies for the quote-bearing pages.
type viewServer struct {
	store        *store.Store
	watchlist    *config.Watchlist
	clock        func() time.Time
	historyLimit int
}

func newViewServer(s *store.Store, wl *config.Watchlist) *viewServer {
	return &viewServer{
		store:        s,
		watchlist:    wl,
		clock:        time.Now,
		historyLimit: DefaultHistoryLimit,
	}
}

// DashboardData feeds the dashboard template.
type DashboardData struct {
	Freshness           FreshnessInfo
	TriggeredAlertCount int
	AlertError          bool
	SectionCounts       map[string]int
}

// ReportData feeds the report template.
type ReportData struct {
	Title     string
	Date      string
	Freshness FreshnessInfo
	Sections  []ReportSection
}

// ReportSection groups report rows by asset class.
type ReportSection struct {
	Name string
	Rows []ReportRow
}

// ReportRow is a single line in a report section.
type ReportRow struct {
	Label     string
	Price     string
	Pct       string
	PctClass  string
	Freshness FreshnessInfo
}

// WatchlistData feeds the watchlist template.
type WatchlistData struct {
	Freshness FreshnessInfo
	Sections  []WatchlistSection
}

// WatchlistSection groups watchlist rows by asset class.
type WatchlistSection struct {
	Name string
	Rows []WatchlistRow
}

// WatchlistRow is a single watchlist entry.
type WatchlistRow struct {
	Label     string
	Source    string
	Symbol    string
	Price     string
	Pct       string
	PctClass  string
	Freshness FreshnessInfo
}

// AssetDetailData feeds the asset detail template.
type AssetDetailData struct {
	Source    string
	Symbol    string
	Name      string
	Label     string
	Price     string
	Currency  string
	Pct       string
	PctClass  string
	Bid       string
	Ask       string
	Freshness FreshnessInfo
	History   []AssetHistoryPoint
}

// AssetHistoryPoint is one history sample for the asset detail chart/table.
type AssetHistoryPoint struct {
	FetchedAt time.Time
	Price     string
}

func (vs *viewServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	now := vs.clock()
	pairs := vs.watchlistPairs()
	snapshots, err := vs.store.LatestSnapshots(pairs)

	freshnessRecords := make([]FreshnessInfo, 0, len(pairs))
	sectionCounts := make(map[string]int)
	for _, p := range pairs {
		rec, ok := snapshots[p]
		if err != nil {
			freshnessRecords = append(freshnessRecords, FreshnessInfo{State: FreshnessUnavailable})
			continue
		}
		if !ok {
			freshnessRecords = append(freshnessRecords, FreshnessInfo{State: FreshnessMissing})
			continue
		}
		freshnessRecords = append(freshnessRecords, AssessFreshness(rec.FetchedAt, now, nil))
		sectionCounts[p.Source]++
	}

	alertCount, alertErr := vs.triggeredAlertCount()
	data := DashboardData{
		Freshness:           NewestFreshness(freshnessRecords),
		TriggeredAlertCount: alertCount,
		AlertError:          alertErr,
		SectionCounts:       sectionCounts,
	}

	vs.render(w, r, "inicio", "Inicio", Dashboard(data))
}

func (vs *viewServer) reportCurrentHandler(w http.ResponseWriter, r *http.Request) {
	now := vs.clock()
	data := vs.buildReport("Reporte", now, "")
	data = vs.fillReportCurrent(data, now)
	vs.render(w, r, "reporte", "Reporte", Report(data))
}

func (vs *viewServer) reportDatedHandler(w http.ResponseWriter, r *http.Request) {
	dateStr := r.PathValue("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Fecha inválida: usá el formato AAAA-MM-DD", http.StatusBadRequest)
		return
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, arLocation)
	end := start.Add(24*time.Hour - time.Millisecond)

	data := vs.buildReport(fmt.Sprintf("Reporte del %s", dateStr), vs.clock(), dateStr)
	data = vs.fillReportForDate(data, start, end)

	if len(data.Sections) == 0 || allRowsMissing(data.Sections) {
		w.WriteHeader(http.StatusNotFound)
		vs.render(w, r, "reporte", "No encontrado", ErrorNotFound("No hay datos para esa fecha."))
		return
	}

	vs.render(w, r, "reporte", data.Title, Report(data))
}

func (vs *viewServer) watchlistHandler(w http.ResponseWriter, r *http.Request) {
	now := vs.clock()
	pairs := vs.watchlistPairs()
	snapshots, err := vs.store.LatestSnapshots(pairs)

	freshnessRecords := make([]FreshnessInfo, 0, len(pairs))
	var sections []WatchlistSection

	if rows, f := watchlistSectionRows(vs, "Acciones USA", vs.watchlist.USA,
		func(a config.Asset) (string, store.WebQuoteKey) {
			return a.Label, store.WebQuoteKey{Source: "yahoo", Symbol: a.Symbol}
		}, snapshots, err, now); len(rows) > 0 {
		freshnessRecords = append(freshnessRecords, f...)
		sections = append(sections, WatchlistSection{Name: "Acciones USA", Rows: rows})
	}

	if rows, f := watchlistSectionRows(vs, "Dólares", vs.watchlist.Dolares,
		func(d string) (string, store.WebQuoteKey) {
			return d, store.WebQuoteKey{Source: "dolarapi", Symbol: d}
		}, snapshots, err, now); len(rows) > 0 {
		freshnessRecords = append(freshnessRecords, f...)
		sections = append(sections, WatchlistSection{Name: "Dólares", Rows: rows})
	}

	if rows, f := watchlistSectionRows(vs, "Bonos", vs.watchlist.Bonos,
		func(b string) (string, store.WebQuoteKey) {
			return b, store.WebQuoteKey{Source: "data912", Symbol: b}
		}, snapshots, err, now); len(rows) > 0 {
		freshnessRecords = append(freshnessRecords, f...)
		sections = append(sections, WatchlistSection{Name: "Bonos", Rows: rows})
	}

	if rows, f := watchlistSectionRows(vs, "Cripto", vs.watchlist.Cripto,
		func(c config.CryptoAsset) (string, store.WebQuoteKey) {
			return c.Label, store.WebQuoteKey{Source: "coingecko", Symbol: c.ID}
		}, snapshots, err, now); len(rows) > 0 {
		freshnessRecords = append(freshnessRecords, f...)
		sections = append(sections, WatchlistSection{Name: "Cripto", Rows: rows})
	}

	data := WatchlistData{
		Freshness: NewestFreshness(freshnessRecords),
		Sections:  sections,
	}
	vs.render(w, r, "activos", "Activos", Watchlist(data))
}

func (vs *viewServer) assetDetailHandler(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")
	symbol := r.PathValue("symbol")

	now := vs.clock()
	quote, found, err := vs.store.LatestQuote(source, symbol)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		vs.render(w, r, "activos", "No disponible", ErrorNotFound("No se pudo leer el activo."))
		return
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		vs.render(w, r, "activos", "No encontrado", ErrorNotFound("El activo solicitado no existe."))
		return
	}

	from := now.AddDate(0, -3, 0)
	history, histErr := vs.store.QuoteHistory(source, symbol, from, now, DefaultAssetHistoryLimit)
	freshness := AssessFreshness(quote.FetchedAt, now, histErr)
	if histErr != nil {
		freshness = FreshnessInfo{State: FreshnessUnavailable}
	}

	points := make([]AssetHistoryPoint, 0, len(history))
	for _, h := range history {
		points = append(points, AssetHistoryPoint{
			FetchedAt: h.FetchedAt,
			Price:     FormatPrice(h.Price, h.Currency),
		})
	}

	data := AssetDetailData{
		Source:    quote.Source,
		Symbol:    quote.Symbol,
		Name:      quote.Name,
		Label:     assetLabel(vs.watchlist, source, symbol),
		Price:     FormatPrice(quote.Price, quote.Currency),
		Currency:  quote.Currency,
		Pct:       FormatPercent(quote.ChangePct),
		PctClass:  PercentClass(quote.ChangePct),
		Bid:       FormatPrice(quote.Bid, quote.Currency),
		Ask:       FormatPrice(quote.Ask, quote.Currency),
		Freshness: freshness,
		History:   points,
	}

	vs.render(w, r, "activos", data.Label, AssetDetail(data))
}

func (vs *viewServer) render(w http.ResponseWriter, r *http.Request, current, title string, body templ.Component) {
	if RequestIsHX(r) {
		_ = Fragment(body).Render(r.Context(), w)
		return
	}
	_ = Shell(title, current, body).Render(r.Context(), w)
}

func (vs *viewServer) triggeredAlertCount() (int, bool) {
	alerts, err := vs.store.History(vs.historyLimit)
	if err != nil {
		return 0, true
	}
	return len(alerts), false
}

func (vs *viewServer) watchlistPairs() []store.WebQuoteKey {
	pairs := make([]store.WebQuoteKey, 0,
		len(vs.watchlist.USA)+len(vs.watchlist.Dolares)+len(vs.watchlist.Bonos)+len(vs.watchlist.Cripto))
	for _, a := range vs.watchlist.USA {
		pairs = append(pairs, store.WebQuoteKey{Source: "yahoo", Symbol: a.Symbol})
	}
	for _, d := range vs.watchlist.Dolares {
		pairs = append(pairs, store.WebQuoteKey{Source: "dolarapi", Symbol: d})
	}
	for _, b := range vs.watchlist.Bonos {
		pairs = append(pairs, store.WebQuoteKey{Source: "data912", Symbol: b})
	}
	for _, c := range vs.watchlist.Cripto {
		pairs = append(pairs, store.WebQuoteKey{Source: "coingecko", Symbol: c.ID})
	}
	return pairs
}

func watchlistSectionRows[T any](vs *viewServer, name string, items []T,
	keyFn func(T) (string, store.WebQuoteKey),
	snapshots map[store.WebQuoteKey]store.QuoteRecord, err error, now time.Time) ([]WatchlistRow, []FreshnessInfo) {

	rows := make([]WatchlistRow, 0, len(items))
	freshness := make([]FreshnessInfo, 0, len(items))
	for _, it := range items {
		label, key := keyFn(it)
		row, f := vs.watchlistRow(label, key, snapshots, err, now)
		rows = append(rows, row)
		freshness = append(freshness, f)
	}
	return rows, freshness
}

func (vs *viewServer) watchlistRow(label string, key store.WebQuoteKey, snapshots map[store.WebQuoteKey]store.QuoteRecord, err error, now time.Time) (WatchlistRow, FreshnessInfo) {
	if err != nil {
		return WatchlistRow{
			Label: label, Source: key.Source, Symbol: key.Symbol,
			Freshness: FreshnessInfo{State: FreshnessUnavailable},
		}, FreshnessInfo{State: FreshnessUnavailable}
	}
	rec, ok := snapshots[key]
	if !ok {
		return WatchlistRow{
			Label: label, Source: key.Source, Symbol: key.Symbol,
			Freshness: FreshnessInfo{State: FreshnessMissing},
		}, FreshnessInfo{State: FreshnessMissing}
	}
	freshness := AssessFreshness(rec.FetchedAt, now, nil)
	return WatchlistRow{
		Label:     label,
		Source:    key.Source,
		Symbol:    key.Symbol,
		Price:     FormatPrice(rec.Price, rec.Currency),
		Pct:       FormatPercent(rec.ChangePct),
		PctClass:  PercentClass(rec.ChangePct),
		Freshness: freshness,
	}, freshness
}

func (vs *viewServer) buildReport(title string, now time.Time, date string) ReportData {
	return ReportData{
		Title: title,
		Date:  date,
	}
}

func (vs *viewServer) fillReportForDate(data ReportData, start, end time.Time) ReportData {
	now := vs.clock()
	freshnessRecords := make([]FreshnessInfo, 0)

	usaRows := make([]ReportRow, 0, len(vs.watchlist.USA))
	for _, a := range vs.watchlist.USA {
		row, f := reportRowForPair(a.Label, "yahoo", a.Symbol, start, end, vs.store, now)
		usaRows = append(usaRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}
	dolarRows := make([]ReportRow, 0, len(vs.watchlist.Dolares))
	for _, d := range vs.watchlist.Dolares {
		row, f := reportRowForPair(d, "dolarapi", d, start, end, vs.store, now)
		dolarRows = append(dolarRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}
	bonoRows := make([]ReportRow, 0, len(vs.watchlist.Bonos))
	for _, b := range vs.watchlist.Bonos {
		row, f := reportRowForPair(b, "data912", b, start, end, vs.store, now)
		bonoRows = append(bonoRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}
	cryptoRows := make([]ReportRow, 0, len(vs.watchlist.Cripto))
	for _, c := range vs.watchlist.Cripto {
		row, f := reportRowForPair(c.Label, "coingecko", c.ID, start, end, vs.store, now)
		cryptoRows = append(cryptoRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}

	data.Sections = buildReportSections(usaRows, dolarRows, bonoRows, cryptoRows)
	data.Freshness = NewestFreshness(freshnessRecords)
	return data
}

func (vs *viewServer) fillReportCurrent(data ReportData, now time.Time) ReportData {
	pairs := vs.watchlistPairs()
	snapshots, err := vs.store.LatestSnapshots(pairs)

	freshnessRecords := make([]FreshnessInfo, 0, len(pairs))

	usaRows := make([]ReportRow, 0, len(vs.watchlist.USA))
	for _, a := range vs.watchlist.USA {
		row, f := reportCurrentRow(a.Label, store.WebQuoteKey{Source: "yahoo", Symbol: a.Symbol}, snapshots, err, now)
		usaRows = append(usaRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}
	dolarRows := make([]ReportRow, 0, len(vs.watchlist.Dolares))
	for _, d := range vs.watchlist.Dolares {
		row, f := reportCurrentRow(d, store.WebQuoteKey{Source: "dolarapi", Symbol: d}, snapshots, err, now)
		dolarRows = append(dolarRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}
	bonoRows := make([]ReportRow, 0, len(vs.watchlist.Bonos))
	for _, b := range vs.watchlist.Bonos {
		row, f := reportCurrentRow(b, store.WebQuoteKey{Source: "data912", Symbol: b}, snapshots, err, now)
		bonoRows = append(bonoRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}
	cryptoRows := make([]ReportRow, 0, len(vs.watchlist.Cripto))
	for _, c := range vs.watchlist.Cripto {
		row, f := reportCurrentRow(c.Label, store.WebQuoteKey{Source: "coingecko", Symbol: c.ID}, snapshots, err, now)
		cryptoRows = append(cryptoRows, row)
		freshnessRecords = append(freshnessRecords, f)
	}

	data.Sections = buildReportSections(usaRows, dolarRows, bonoRows, cryptoRows)
	data.Freshness = NewestFreshness(freshnessRecords)
	return data
}

func reportCurrentRow(label string, key store.WebQuoteKey, snapshots map[store.WebQuoteKey]store.QuoteRecord, err error, now time.Time) (ReportRow, FreshnessInfo) {
	if err != nil {
		return ReportRow{
			Label: label, Price: "—", Pct: "—", PctClass: "negative",
			Freshness: FreshnessInfo{State: FreshnessUnavailable},
		}, FreshnessInfo{State: FreshnessUnavailable}
	}
	rec, ok := snapshots[key]
	if !ok {
		return ReportRow{
			Label: label, Price: "—", Pct: "—", PctClass: "positive",
			Freshness: FreshnessInfo{State: FreshnessMissing},
		}, FreshnessInfo{State: FreshnessMissing}
	}
	freshness := AssessFreshness(rec.FetchedAt, now, nil)
	return ReportRow{
		Label:     label,
		Price:     FormatPrice(rec.Price, rec.Currency),
		Pct:       FormatPercent(rec.ChangePct),
		PctClass:  PercentClass(rec.ChangePct),
		Freshness: freshness,
	}, freshness
}

func buildReportSections(usa, dolar, bono, crypto []ReportRow) []ReportSection {
	var sections []ReportSection
	if len(usa) > 0 {
		sections = append(sections, ReportSection{Name: "Acciones USA", Rows: sortReportRowsCopy(usa)})
	}
	if len(dolar) > 0 {
		sections = append(sections, ReportSection{Name: "Dólares", Rows: sortReportRowsCopy(dolar)})
	}
	if len(bono) > 0 {
		sections = append(sections, ReportSection{Name: "Bonos", Rows: sortReportRowsCopy(bono)})
	}
	if len(crypto) > 0 {
		sections = append(sections, ReportSection{Name: "Cripto", Rows: sortReportRowsCopy(crypto)})
	}
	return sections
}

func reportRowForPair(label string, source, symbol string, start, end time.Time, s *store.Store, now time.Time) (ReportRow, FreshnessInfo) {
	history, err := s.QuoteHistory(source, symbol, start, end, 100)
	if err != nil {
		return ReportRow{
			Label: label, Price: "—", Pct: "—", PctClass: "negative",
			Freshness: FreshnessInfo{State: FreshnessUnavailable},
		}, FreshnessInfo{State: FreshnessUnavailable}
	}
	if len(history) == 0 {
		return ReportRow{
			Label: label, Price: "—", Pct: "—", PctClass: "positive",
			Freshness: FreshnessInfo{State: FreshnessMissing},
		}, FreshnessInfo{State: FreshnessMissing}
	}
	rec := history[len(history)-1]
	freshness := AssessFreshness(rec.FetchedAt, now, nil)
	return ReportRow{
		Label:     label,
		Price:     FormatPrice(rec.Price, rec.Currency),
		Pct:       FormatPercent(rec.ChangePct),
		PctClass:  PercentClass(rec.ChangePct),
		Freshness: freshness,
	}, freshness
}

func allRowsMissing(sections []ReportSection) bool {
	for _, sec := range sections {
		for _, row := range sec.Rows {
			if row.Freshness.State != FreshnessMissing {
				return false
			}
		}
	}
	return true
}

func assetLabel(wl *config.Watchlist, source, symbol string) string {
	switch source {
	case "yahoo":
		return wl.LabelForSymbol(symbol)
	case "coingecko":
		return wl.CryptoLabel(symbol)
	}
	return symbol
}

func sortReportRowsCopy(rows []ReportRow) []ReportRow {
	out := make([]ReportRow, len(rows))
	copy(out, rows)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Label) < strings.ToLower(out[j].Label)
	})
	return out
}
