package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

type selector struct {
	ID       string
	Number   int
	Selected string
}

type outputData struct {
	Effects   []string
	Selectors []selector
	Output    template.HTML
	Warning   string
}

var potions map[string][]string

func outputPotions(values []string) (template.HTML, string) {
	var input []string
	var warning []string

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == "Select an Effect" {
			continue
		}
		if _, ok := potions[value]; !ok {
			warning = append(warning, value)
		} else if value != "Select an Effect" && !contains(input, value) {
			input = append(input, value)
		}
	}

	var finalWarning string
	if len(warning) > 0 {
		finalWarning = "The following effects were not found: " + strings.Join(warning, ", ")
	}

	if len(input) == 0 {
		return "", finalWarning
	}

	possibleItems := make(map[string][]string)
	for _, effect := range input {
		for _, ingredient := range potions[effect] {
			possibleItems[ingredient] = append(possibleItems[ingredient], effect)
		}
	}

	itemLevels := make([]map[string][]string, len(input))
	for index := range itemLevels {
		itemLevels[index] = map[string][]string{}
	}

	var items []string
	for item := range possibleItems {
		items = append(items, item)
	}
	sort.Strings(items)

	for _, item := range items {
		key := possibleItems[item]
		level := len(input) - len(key)
		effects := strings.Join(key, " + ")
		itemLevels[level][effects] = append(itemLevels[level][effects], item)
	}

	var output strings.Builder
	for index, category := range itemLevels {
		if len(category) == 0 {
			continue
		}

		matchingEffects := len(input) - index
		titleEffects := strconv.Itoa(matchingEffects)
		if index == 0 && len(input) != 1 {
			titleEffects = "All"
		}

		output.WriteString(`<div class="resultCard"><h1>Items Matching `)
		output.WriteString(titleEffects)
		output.WriteString(` Effect`)
		if matchingEffects != 1 {
			output.WriteString(`s`)
		}
		output.WriteString(`</h1>`)

		var effectsList []string
		for effects := range category {
			effectsList = append(effectsList, effects)
		}
		sort.Strings(effectsList)

		for _, effects := range effectsList {
			sort.Strings(category[effects])
			output.WriteString(`<h2>`)
			output.WriteString(template.HTMLEscapeString(effects))
			output.WriteString(`</h2><div class="ingredientList"><div class="ingredient">`)
			for itemIndex, item := range category[effects] {
				if itemIndex > 0 {
					output.WriteString(`</div><div class="ingredient">`)
				}
				output.WriteString(template.HTMLEscapeString(item))
			}
			output.WriteString(`</div></div>`)
		}

		output.WriteString(`</div>`)
	}

	return template.HTML(output.String()), finalWarning
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func effectNames() []string {
	var effects []string
	for effect := range potions {
		effects = append(effects, effect)
	}
	sort.Strings(effects)
	return append([]string{"Select an Effect"}, effects...)
}

func requestedEffects(query url.Values) []string {
	type numberedValue struct {
		number int
		value  string
	}

	var found []numberedValue
	for key, values := range query {
		if !strings.HasPrefix(key, "I") {
			continue
		}
		number, err := strconv.Atoi(strings.TrimPrefix(key, "I"))
		if err != nil || number < 1 {
			continue
		}
		value := ""
		if len(values) > 0 {
			value = values[0]
		}
		found = append(found, numberedValue{number: number, value: value})
	}

	if len(found) == 0 {
		return make([]string, 4)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].number < found[j].number })

	values := make([]string, len(found))
	for i, entry := range found {
		values[i] = entry.value
	}
	return values
}

func buildSelectors(values []string) []selector {
	selectors := make([]selector, len(values))
	for i, value := range values {
		selectors[i] = selector{ID: "I" + strconv.Itoa(i+1), Number: i + 1, Selected: value}
	}
	return selectors
}

func main() {
	jsonData, err := os.ReadFile("data/potions.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(jsonData), &potions)

	if err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.ParseFiles("views/main.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		values := requestedEffects(r.URL.Query())

		output, warning := outputPotions(values)
		out := outputData{
			Effects:   effectNames(),
			Selectors: buildSelectors(values),
			Output:    output,
			Warning:   warning,
		}
		if err := tmpl.Execute(w, out); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(potions)
	})

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":80", nil)
}
