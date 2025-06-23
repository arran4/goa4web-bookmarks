package a4webbm

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRenderBookmarksTemplates(t *testing.T) {
	list := ParseBookmarks(defaultBookmarks)
	funcs := template.FuncMap{
		"bookmarkPages": func() ([]*BookmarkPage, error) { return list[0].Pages, nil },
		"bookmarkTabs": func() ([]TabInfo, error) {
			last := ""
			if len(list[0].Pages) > 0 {
				last = list[0].Pages[len(list[0].Pages)-1].Sha()
			}
			return []TabInfo{{Index: 0, Name: list[0].Name, IndexName: "Main", Href: "/", LastPageSha: last}}, nil
		},
		"tabName":   func() string { return "Main" },
		"add1":      func(i int) int { return i + 1 },
		"OAuth2URL": func() string { return "" },
		"now":       func() time.Time { return time.Now() },
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(os.DirFS("./templates"), "head.gohtml", "tail.gohtml", "bookmarksMinePage.gohtml"))
	var buf bytes.Buffer
	data := struct {
		Title         string
		AutoRefresh   bool
		UseCssColumns bool
		UserRef       string
		NoFooter      bool
	}{Title: "t", UseCssColumns: false, UserRef: "u"}
	if err := tmpl.ExecuteTemplate(&buf, "bookmarksMinePage.gohtml", data); err != nil {
		t.Fatalf("render err: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<a href") {
		t.Fatalf("expected links in output: %s", out)
	}
}
