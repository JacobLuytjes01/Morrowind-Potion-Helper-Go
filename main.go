package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type selection struct {
	Value      string
	SelectedBy int
}

type outputData struct {
	Selections []selection
	Output     template.HTML
	Warning    string
}

var potions map[string][]string

func outputPotions(I1 string, I2 string, I3 string, I4 string) (template.HTML, string) {
	var input []string
	var warning []string

	for _, value := range []string{I1, I2, I3, I4} {
		value = strings.TrimSpace(value)
		if value == "" {
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

	possibleItems := map[string]bool{}
	for _, effect := range input {
		for _, ingredient := range potions[effect] {
			possibleItems[ingredient] = true
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
		var key []string
		level := len(input)
		for _, effect := range input {
			if contains(potions[effect], item) {
				key = append(key, effect)
				level--
			}
		}

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

func buildSelections(I1 string, I2 string, I3 string, I4 string) []selection {
	var effects []string
	for effect := range potions {
		effects = append(effects, effect)
	}
	sort.Strings(effects)

	selections := make([]selection, 0, len(effects))
	for _, effect := range effects {
		selectedBy := 0
		switch effect {
		case I1:
			selectedBy = 1
		case I2:
			selectedBy = 2
		case I3:
			selectedBy = 3
		case I4:
			selectedBy = 4
		}

		selections = append(selections, selection{
			Value:      effect,
			SelectedBy: selectedBy,
		})
	}

	return selections
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
		queryParams := r.URL.Query()

		output, warning := outputPotions(queryParams.Get("I1"), queryParams.Get("I2"), queryParams.Get("I3"), queryParams.Get("I4"))
		out := outputData{
			Selections: buildSelections(queryParams.Get("I1"), queryParams.Get("I2"), queryParams.Get("I3"), queryParams.Get("I4")),
			Output:  output,
			Warning: warning,
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
