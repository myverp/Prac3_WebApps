package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"

	"webapps/internal/site"
)

var defaults = map[string]float64{
	"pc":     5,
	"price":  7,
	"delta":  5,
	"sigma1": 1,
	"sigma2": 0.25,
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Prac3 running on http://127.0.0.1:8083")
	if err := http.ListenAndServe(":8083", nil); err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	values := map[string]float64{}
	for k, v := range defaults {
		values[k] = v
	}
	for _, name := range []string{"pc", "price", "delta", "sigma1", "sigma2"} {
		if raw := r.URL.Query().Get(name); raw != "" {
			values[name] = parseFloat(raw, values[name])
		}
	}

	before := scenario(values["pc"], values["price"], values["delta"], values["sigma1"])
	after := scenario(values["pc"], values["price"], values["delta"], values["sigma2"])

	rows := [][]string{
		{"До покращення", fmtNumber(before.sharePct, 0) + "%", fmtNumber(before.sold, 2), fmtNumber(before.revenue, 2), fmtNumber(before.penalty, 2)},
		{"Після покращення", fmtNumber(after.sharePct, 0) + "%", fmtNumber(after.sold, 2), fmtNumber(after.revenue, 2), fmtNumber(after.penalty, 2)},
	}

	data := site.PageData{
		Title:      "Практична робота 3",
		Practice:   "Практична робота 3",
		Breadcrumb: "Завдання 3",
		Lead:       "Калькулятор показує, як зміна точності прогнозу впливає на частку потужності в допустимому інтервалі, дохід та штраф.",
		Tags:       []string{"Нормальний розподіл", "До і після", "Дохід і штраф"},
		Highlights: []string{
			"Контрольний приклад використовує Pc = 5 МВт, σ1 = 1 МВт та σ2 = 0,25 МВт.",
			"Частка ймовірності округлюється до цілого відсотка, як у наведеному в завданні прикладі.",
			"Розрахунок виконується на сервері Go без клієнтського скрипта.",
		},
		Fields: []site.Field{
			numericField("Середня потужність Pc, МВт", "pc", values["pc"], "0.01"),
			numericField("Ціна електроенергії, грн/кВт·год", "price", values["price"], "0.01"),
			numericField("Допустиме відхилення, %", "delta", values["delta"], "0.01"),
			numericField("Сигма до покращення, МВт", "sigma1", values["sigma1"], "0.01"),
			numericField("Сигма після покращення, МВт", "sigma2", values["sigma2"], "0.01"),
		},
		Metrics: []site.Metric{
			{Label: "Частка до", Value: fmtNumber(before.sharePct, 0) + "%"},
			{Label: "Частка після", Value: fmtNumber(after.sharePct, 0) + "%"},
			{Label: "Чистий прибуток до", Value: fmtNumber(before.net, 2) + " тис. грн"},
			{Label: "Чистий прибуток після", Value: fmtNumber(after.net, 2) + " тис. грн"},
			{Label: "Приріст", Value: fmtNumber(after.net-before.net, 2) + " тис. грн"},
			{Label: "Зменшення штрафу", Value: fmtNumber(before.penalty-after.penalty, 2) + " тис. грн"},
		},
		Sections: []site.Section{
			{Title: "Розрахунок", HTML: site.Table([]string{"Сценарій", "Допустима частка", "Енергія, МВт·год", "Дохід, тис. грн", "Штраф, тис. грн"}, rows)},
			{Title: "Примітка", HTML: template.HTML(`<p>Для контрольного прикладу сторінка відтворює ті самі вихідні значення, що і у PDF. Через це результат легко порівняти з розв'язком у зошиті.</p>`)},
		},
		Footer: "Висновок: підвищення точності прогнозу зменшує штраф та підвищує чистий прибуток сонячної електростанції.",
	}

	if err := site.Render(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type scenarioResult struct {
	share    float64
	sharePct float64
	sold     float64
	revenue  float64
	penalty  float64
	net      float64
}

func scenario(pc, price, delta, sigma float64) scenarioResult {
	lower := pc * (1 - delta/100)
	upper := pc * (1 + delta/100)
	share := normalProbabilityBetween(pc, sigma, lower, upper)
	shareRounded := math.Round(share*100) / 100
	sharePct := math.Round(shareRounded * 100)
	sold := pc * 24 * shareRounded
	rejected := pc * 24 * (1 - shareRounded)
	revenue := sold * price
	penalty := rejected * price
	return scenarioResult{
		share:    shareRounded,
		sharePct: sharePct,
		sold:     sold,
		revenue:  revenue,
		penalty:  penalty,
		net:      revenue - penalty,
	}
}

func normalProbabilityBetween(mean, sigma, lower, upper float64) float64 {
	lo := math.Min(lower, upper)
	hi := math.Max(lower, upper)
	return normalCdf(hi, mean, sigma) - normalCdf(lo, mean, sigma)
}

func normalCdf(x, mean, sigma float64) float64 {
	if sigma <= 0 {
		if x < mean {
			return 0
		}
		return 1
	}
	return 0.5 * (1 + math.Erf((x-mean)/(sigma*math.Sqrt2)))
}

func numericField(label, name string, value float64, step string) site.Field {
	return site.Field{Label: label, Name: name, Type: "number", Step: step, Value: fmtNumber(value, 2)}
}

func parseFloat(raw string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64); err == nil {
		return v
	}
	return fallback
}

func fmtNumber(value float64, digits int) string {
	return strings.ReplaceAll(fmt.Sprintf("%.*f", digits, value), ".", ",")
}
