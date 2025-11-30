package main

import (
	"html/template"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/antonmedv/gitmal/pkg/templates"
)

var themeStyles = map[string]string{
	"abap":                 "light",
	"algol":                "light",
	"arduino":              "light",
	"autumn":               "light",
	"average":              "dark",
	"base16-snazzy":        "dark",
	"borland":              "light",
	"bw":                   "light",
	"catppuccin-frappe":    "dark",
	"catppuccin-latte":     "light",
	"catppuccin-macchiato": "dark",
	"catppuccin-mocha":     "dark",
	"colorful":             "light",
	"doom-one":             "dark",
	"doom-one2":            "dark",
	"dracula":              "dark",
	"emacs":                "light",
	"evergarden":           "dark",
	"friendly":             "light",
	"fruity":               "dark",
	"github-dark":          "dark",
	"github":               "light",
	"gruvbox-light":        "light",
	"gruvbox":              "dark",
	"hrdark":               "dark",
	"igor":                 "light",
	"lovelace":             "light",
	"manni":                "light",
	"modus-operandi":       "light",
	"modus-vivendi":        "dark",
	"monokai":              "dark",
	"monokailight":         "light",
	"murphy":               "light",
	"native":               "dark",
	"nord":                 "dark",
	"nordic":               "dark",
	"onedark":              "dark",
	"onesenterprise":       "dark",
	"paraiso-dark":         "dark",
	"paraiso-light":        "light",
	"pastie":               "light",
	"perldoc":              "light",
	"pygments":             "light",
	"rainbow_dash":         "light",
	"rose-pine-dawn":       "light",
	"rose-pine-moon":       "dark",
	"rose-pine":            "dark",
	"rpgle":                "dark",
	"rrt":                  "dark",
	"solarized-dark":       "dark",
	"solarized-dark256":    "dark",
	"solarized-light":      "light",
	"swapoff":              "dark",
	"tango":                "light",
	"tokyonight-day":       "light",
	"tokyonight-moon":      "dark",
	"tokyonight-night":     "dark",
	"tokyonight-storm":     "dark",
	"trac":                 "light",
	"vim":                  "dark",
	"vs":                   "light",
	"vulcan":               "dark",
	"witchhazel":           "dark",
	"xcode-dark":           "dark",
	"xcode":                "light",
}

func previewThemes() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		names := make([]string, 0, len(themeStyles))
		for name := range themeStyles {
			names = append(names, name)
		}
		sort.Strings(names)

		sampleLang := "javascript"
		sampleCode := `function fib(n) {
    if (n <= 1) {
        return n;
    }
    return fib(n - 1) + fib(n - 2);
}

// Print n Fibonacci numbers.
const n = 10;

for (let i = 0; i < n; i++) {
    console.log(fib(i));
}`

		formatter := html.New(
			html.WithClasses(false),
		)

		// Generate cards
		cards := make([]templates.PreviewCard, 0, len(names))
		for _, theme := range names {
			style := styles.Get(theme)
			if style == nil {
				continue
			}
			lexer := lexers.Get(sampleLang)
			if lexer == nil {
				continue
			}
			it, err := lexer.Tokenise(nil, sampleCode)
			if err != nil {
				continue
			}
			var sb strings.Builder
			if err := formatter.Format(&sb, style, it); err != nil {
				continue
			}
			cards = append(cards, templates.PreviewCard{
				Name: theme,
				Tone: themeStyles[theme],
				HTML: template.HTML(sb.String()),
			})
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = templates.PreviewTemplate.Execute(w, templates.PreviewParams{
			Count:  len(cards),
			Themes: cards,
		})
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	addr := ln.Addr().String()
	echo("Preview themes at http://" + addr)

	if err := http.Serve(ln, handler); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
