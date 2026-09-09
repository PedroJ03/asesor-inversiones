package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/PedroJ03/asesor-inversiones/internal/config"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

// DefaultAlertsHistoryLimit is the number of recent alert snapshots shown on
// the alerts page.
const DefaultAlertsHistoryLimit = 50

// ruleServer holds the dependencies for alert-rule handlers.
type ruleServer struct {
	store     *store.Store
	watchlist *config.Watchlist
	clock     func() time.Time
}

func newRuleServer(s *store.Store, wl *config.Watchlist) *ruleServer {
	return &ruleServer{
		store:     s,
		watchlist: wl,
		clock:     time.Now,
	}
}

// RuleView is a presentation-friendly alert rule.
type RuleView struct {
	ID            int64
	Source        string
	Symbol        string
	Label         string
	Kind          string
	Direction     string
	Threshold     string
	BaselinePrice string
	Enabled       bool
	EnabledClass  string
	State         string
	StateClass    string
}

// AlertView is a presentation-friendly triggered alert snapshot.
type AlertView struct {
	Source      string
	Symbol      string
	Kind        string
	Threshold   string
	TriggeredAt string
}

// WatchlistOption is one selectable asset in the rule creation form.
type WatchlistOption struct {
	Source string
	Symbol string
	Label  string
}

// RuleFormData carries the rule creation form state and any validation error.
type RuleFormData struct {
	Source    string
	Symbol    string
	Kind      string
	Direction string
	Threshold string
	Error     string
	Options   []WatchlistOption
}

// AlertsData feeds the alerts page template.
type AlertsData struct {
	Rules   []RuleView
	History []AlertView
	Form    RuleFormData
	Error   string
}

func (rs *ruleServer) alertsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := rs.buildAlertsData(r, rs.formDataFromQuery(r))
	if err != nil {
		data = AlertsData{Error: ruleListErrorMessage(err)}
	}
	rs.render(w, r, "alertas", "Alertas", Alerts(data))
}

func (rs *ruleServer) ruleCreateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		rs.renderFormError(w, r, RuleFormData{Error: "El formulario no pudo leerse."})
		return
	}

	form := rs.parseRuleForm(r)
	if form.Error != "" {
		rs.renderFormError(w, r, form)
		return
	}

	baseline, err := rs.baselineFor(form.Source, form.Symbol)
	if err != nil {
		form.Error = err.Error()
		rs.renderFormError(w, r, form)
		return
	}

	threshold, err := strconv.ParseFloat(normalizeThreshold(form.Threshold), 64)
	if err != nil || threshold <= 0 {
		form.Error = "El umbral debe ser un número positivo."
		rs.renderFormError(w, r, form)
		return
	}

	_, err = rs.store.CreateRule(form.Source, form.Symbol, form.Kind, form.Direction, threshold, baseline, rs.clock())
	if err != nil {
		form.Error = classifyRuleError(err)
		rs.renderFormError(w, r, form)
		return
	}

	if RequestIsHX(r) {
		data, err := rs.buildAlertsData(r, RuleFormData{})
		if err != nil {
			data = AlertsData{Error: ruleListErrorMessage(err)}
		}
		_ = Fragment(Alerts(data)).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/alertas", http.StatusSeeOther)
}

func (rs *ruleServer) ruleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := rs.parseID(r)
	if !ok {
		rs.renderActionError(w, r, "Identificador de regla inválido.")
		return
	}

	// The PUT endpoint toggles the rule's enabled flag. This keeps the
	// armed/triggered state intact so a disabled rule is not misclassified as
	// triggered.
	enabled, err := parseEnabledValue(r.FormValue("enabled"))
	if err != nil {
		rs.renderActionError(w, r, "Estado inválido.")
		return
	}

	if err := rs.store.SetRuleEnabled(id, enabled); err != nil {
		rs.renderActionError(w, r, classifyRuleError(err))
		return
	}

	if RequestIsHX(r) {
		data, err := rs.buildAlertsData(r, RuleFormData{})
		if err != nil {
			data = AlertsData{Error: ruleListErrorMessage(err)}
		}
		_ = Fragment(Alerts(data)).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/alertas", http.StatusSeeOther)
}

func (rs *ruleServer) ruleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := rs.parseID(r)
	if !ok {
		rs.renderActionError(w, r, "Identificador de regla inválido.")
		return
	}

	if err := rs.store.RemoveRule(id); err != nil {
		rs.renderActionError(w, r, classifyRuleError(err))
		return
	}

	if RequestIsHX(r) {
		data, err := rs.buildAlertsData(r, RuleFormData{})
		if err != nil {
			data = AlertsData{Error: ruleListErrorMessage(err)}
		}
		_ = Fragment(Alerts(data)).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/alertas", http.StatusSeeOther)
}

func (rs *ruleServer) ruleActionFallbackHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		rs.renderActionError(w, r, "El formulario no pudo leerse.")
		return
	}

	action := strings.TrimSpace(r.FormValue("action"))
	switch action {
	case "deshabilitar":
		// Disabling a rule sets enabled=0 while preserving its armed/triggered
		// state; it is not mapped to the triggered state hack anymore.
		rs.setRuleEnabled(w, r, false)
	case "habilitar":
		// Re-enabling a rule sets enabled=1, making it active again without
		// changing its armed/triggered state.
		rs.setRuleEnabled(w, r, true)
	case "eliminar":
		rs.deleteRule(w, r)
	default:
		rs.renderActionError(w, r, "Acción no reconocida.")
	}
}

func (rs *ruleServer) setRuleEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	id, ok := rs.parseID(r)
	if !ok {
		rs.renderActionError(w, r, "Identificador de regla inválido.")
		return
	}

	if err := rs.store.SetRuleEnabled(id, enabled); err != nil {
		rs.renderActionError(w, r, classifyRuleError(err))
		return
	}
	http.Redirect(w, r, "/alertas", http.StatusSeeOther)
}

func (rs *ruleServer) deleteRule(w http.ResponseWriter, r *http.Request) {
	id, ok := rs.parseID(r)
	if !ok {
		rs.renderActionError(w, r, "Identificador de regla inválido.")
		return
	}

	if err := rs.store.RemoveRule(id); err != nil {
		rs.renderActionError(w, r, classifyRuleError(err))
		return
	}
	http.Redirect(w, r, "/alertas", http.StatusSeeOther)
}

func (rs *ruleServer) parseID(r *http.Request) (int64, bool) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (rs *ruleServer) parseRuleForm(r *http.Request) RuleFormData {
	source, symbol := parseAssetValue(r.FormValue("source"))
	form := RuleFormData{
		Source:    source,
		Symbol:    symbol,
		Kind:      strings.TrimSpace(r.FormValue("kind")),
		Direction: strings.TrimSpace(r.FormValue("direction")),
		Threshold: strings.TrimSpace(r.FormValue("threshold")),
		Options:   rs.watchlistOptions(),
	}

	if form.Source == "" || form.Symbol == "" {
		form.Error = "Seleccioná un activo."
		return form
	}
	if !rs.validPair(form.Source, form.Symbol) {
		form.Error = "El activo seleccionado no está en la lista."
		return form
	}
	if form.Kind != "value" && form.Kind != "pct" {
		form.Error = "El tipo de alerta debe ser valor o porcentaje."
		return form
	}
	if form.Direction != "above" && form.Direction != "below" {
		form.Error = "La dirección debe ser mayor o menor que."
		return form
	}
	if form.Threshold == "" {
		form.Error = "Ingresá un umbral."
	}
	return form
}

// formDataFromQuery rebuilds the rule form state from GET query parameters.
// It backs the no-JS-friendly re-render of the form when the kind changes
// (?kind=pct): htmx re-requests the page with the current form values so the
// direction labels match the selected kind without losing input.
func (rs *ruleServer) formDataFromQuery(r *http.Request) RuleFormData {
	q := r.URL.Query()
	if q.Get("source") == "" && q.Get("kind") == "" && q.Get("direction") == "" && q.Get("threshold") == "" {
		return RuleFormData{}
	}
	source, symbol := parseAssetValue(q.Get("source"))
	return RuleFormData{
		Source:    source,
		Symbol:    symbol,
		Kind:      strings.TrimSpace(q.Get("kind")),
		Direction: strings.TrimSpace(q.Get("direction")),
		Threshold: strings.TrimSpace(q.Get("threshold")),
		Options:   rs.watchlistOptions(),
	}
}

func (rs *ruleServer) baselineFor(source, symbol string) (float64, error) {
	quote, found, err := rs.store.LatestQuote(source, symbol)
	if err != nil {
		return 0, fmt.Errorf("no se pudo leer la cotización actual: %v", err)
	}
	if !found {
		return 0, errors.New("no hay cotización actual para ese activo.")
	}
	if quote.Price <= 0 {
		return 0, errors.New("la cotización actual no es válida.")
	}
	return quote.Price, nil
}

func (rs *ruleServer) buildAlertsData(r *http.Request, form RuleFormData) (AlertsData, error) {
	rules, err := rs.store.ListRules(false)
	if err != nil {
		return AlertsData{}, err
	}

	history, err := rs.store.History(DefaultAlertsHistoryLimit)
	if err != nil {
		return AlertsData{}, err
	}

	if form.Options == nil {
		form.Options = rs.watchlistOptions()
	}

	return AlertsData{
		Rules:   rs.ruleViews(rules),
		History: rs.alertViews(history),
		Form:    form,
	}, nil
}

func (rs *ruleServer) ruleViews(rules []store.Rule) []RuleView {
	out := make([]RuleView, 0, len(rules))
	for _, rule := range rules {
		label := assetLabel(rs.watchlist, rule.Source, rule.Symbol)
		kindLabel := "valor"
		if rule.Kind == "pct" {
			kindLabel = "porcentaje"
		}
		directionLabel := "mayor a"
		if rule.Direction == "below" {
			directionLabel = "menor a"
		}
		stateClass := "rule__state--armed"
		if rule.State == "triggered" {
			stateClass = "rule__state--triggered"
		}
		enabledClass := "rule__state--enabled"
		if !rule.Enabled {
			enabledClass = "rule__state--disabled"
		}

		threshold := FormatPrice(rule.Threshold, "USD")
		if rule.Kind == "pct" {
			// The direction is embedded in the threshold label so the rules
			// list reads as one condition: ">= 2,00 % desde u$s 118,50".
			op := "≥"
			if rule.Direction == "below" {
				op = "≤"
			}
			threshold = fmt.Sprintf("%s %s %%", op, formatNumber(rule.Threshold, 2))
		}

		out = append(out, RuleView{
			ID:            rule.ID,
			Source:        rule.Source,
			Symbol:        rule.Symbol,
			Label:         label,
			Kind:          kindLabel,
			Direction:     directionLabel,
			Threshold:     threshold,
			BaselinePrice: FormatPrice(rule.BaselinePrice, "USD"),
			Enabled:       rule.Enabled,
			EnabledClass:  enabledClass,
			State:         rule.State,
			StateClass:    stateClass,
		})
	}
	return out
}

func (rs *ruleServer) alertViews(alerts []store.Alert) []AlertView {
	out := make([]AlertView, 0, len(alerts))
	for _, a := range alerts {
		threshold := FormatPrice(a.Threshold, "USD")
		if a.Kind == "pct" {
			threshold = FormatPercent(a.Threshold)
		}
		out = append(out, AlertView{
			Source:      a.Source,
			Symbol:      a.Symbol,
			Kind:        a.Kind,
			Threshold:   threshold,
			TriggeredAt: a.TriggeredAt.In(arLocation).Format("02/01 15:04"),
		})
	}
	return out
}

func (rs *ruleServer) watchlistOptions() []WatchlistOption {
	var opts []WatchlistOption
	for _, a := range rs.watchlist.USA {
		opts = append(opts, WatchlistOption{Source: "yahoo", Symbol: a.Symbol, Label: a.Label})
	}
	for _, d := range rs.watchlist.Dolares {
		opts = append(opts, WatchlistOption{Source: "dolarapi", Symbol: d, Label: d})
	}
	for _, b := range rs.watchlist.Bonos {
		opts = append(opts, WatchlistOption{Source: "data912", Symbol: b, Label: b})
	}
	for _, c := range rs.watchlist.Cripto {
		opts = append(opts, WatchlistOption{Source: "coingecko", Symbol: c.ID, Label: c.Label})
	}
	return opts
}

func (rs *ruleServer) validPair(source, symbol string) bool {
	for _, opt := range rs.watchlistOptions() {
		if opt.Source == source && opt.Symbol == symbol {
			return true
		}
	}
	return false
}

func (rs *ruleServer) render(w http.ResponseWriter, r *http.Request, current, title string, body templ.Component) {
	if RequestIsHX(r) {
		_ = Fragment(body).Render(r.Context(), w)
		return
	}
	_ = Shell(title, current, body).Render(r.Context(), w)
}

func (rs *ruleServer) renderFormError(w http.ResponseWriter, r *http.Request, form RuleFormData) {
	data, err := rs.buildAlertsData(r, form)
	if err != nil {
		data = AlertsData{Error: ruleListErrorMessage(err), Form: form}
	}
	w.WriteHeader(http.StatusBadRequest)
	rs.render(w, r, "alertas", "Alertas", Alerts(data))
}

func (rs *ruleServer) renderActionError(w http.ResponseWriter, r *http.Request, message string) {
	data, err := rs.buildAlertsData(r, RuleFormData{})
	if err != nil {
		data = AlertsData{Error: ruleListErrorMessage(err)}
	}
	data.Error = message
	w.WriteHeader(http.StatusBadRequest)
	rs.render(w, r, "alertas", "Alertas", Alerts(data))
}

// normalizeThreshold prepares a user-entered threshold for ParseFloat: it
// trims surrounding spaces, strips trivially attached currency markers copied
// along with a price, and accepts "," as the decimal separator.
func normalizeThreshold(raw string) string {
	s := strings.TrimSpace(raw)
	for _, marker := range []string{"u$s", "US$", "USD", "usd", "ARS", "ars", "$"} {
		s = strings.TrimSpace(strings.ReplaceAll(s, marker, ""))
	}
	return strings.ReplaceAll(s, ",", ".")
}

// directionAboveLabel returns the above-direction label for a rule kind.
// Percentage rules compare drift against the captured baseline, so the label
// speaks of movement rather than absolute price.
func directionAboveLabel(kind string) string {
	if kind == "pct" {
		return "Sube al menos"
	}
	return "Mayor o igual que"
}

// directionBelowLabel returns the below-direction label for a rule kind.
func directionBelowLabel(kind string) string {
	if kind == "pct" {
		return "Baja al menos"
	}
	return "Menor o igual que"
}

func parseAssetValue(raw string) (source, symbol string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// parseEnabledValue interprets the enabled form value used by the rule
// toggle endpoints. It only accepts explicit "true" or "false" strings.
func parseEnabledValue(raw string) (bool, error) {
	switch strings.TrimSpace(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New("invalid enabled value")
	}
}

func classifyRuleError(err error) string {
	if errors.Is(err, store.ErrInvalidRule) {
		return "La regla no es válida. Revisá los datos e intentá de nuevo."
	}
	if errors.Is(err, store.ErrDuplicateRule) {
		return "Ya existe una regla igual para ese activo."
	}
	return "No se pudo guardar la regla. Intentá de nuevo."
}

func ruleListErrorMessage(err error) string {
	return "No se pudo cargar el listado de alertas."
}
