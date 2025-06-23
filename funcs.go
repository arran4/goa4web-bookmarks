package a4webbm

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	defaultBookmarks = "Category: Example 1\nhttp://www.google.com.au Google\nColumn\nCategory: Example 2\nhttp://www.google.com.au Google\nhttp://www.google.com.au Google\n"
)

// TabInfo is used by templates to display tab navigation with indexes.
type TabInfo struct {
	Index       int
	Name        string
	IndexName   string
	Href        string
	LastPageSha string
}

func NewFuncs(r *http.Request) template.FuncMap {
	return map[string]any{
		"now": func() time.Time { return time.Now() },
		"firstline": func(s string) string {
			return strings.Split(s, "\n")[0]
		},
		"left": func(i int, s string) string {
			l := len(s)
			if l > i {
				l = i
			}
			return s[:l]
		},
		"OAuth2URL": func() string {
			return Oauth2Config.AuthCodeURL("")
		},
		"bookmarks": func() (string, error) {
			queries := r.Context().Value(ContextValues("queries")).(*Queries)
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			userRef, _ := session.Values["UserRef"].(string)

			bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
					return defaultBookmarks, nil
				default:
					return "", err
				}
			}
			return bookmarks.List.String, nil
		},
		"bookmarkColumns": func() ([]*BookmarkColumn, error) {
			queries := r.Context().Value(ContextValues("queries")).(*Queries)
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			userRef, _ := session.Values["UserRef"].(string)

			bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
			var bookmarkString = defaultBookmarks
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
				default:
					return nil, err
				}
			} else {
				bookmarkString = bookmarks.List.String
			}
			list := ParseBookmarks(bookmarkString)
			if len(list) == 0 || len(list[0].Pages) == 0 {
				return nil, nil
			}
			var cols []*BookmarkColumn
			page := list[0].Pages[0]
			for _, blk := range page.Blocks {
				if blk.HR {
					continue
				}
				cols = append(cols, blk.Columns...)
			}
			return cols, nil
		},
		"bookmarkPages": func() ([]*BookmarkPage, error) {
			queries := r.Context().Value(ContextValues("queries")).(*Queries)
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			userRef, _ := session.Values["UserRef"].(string)

			bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
			var bookmarkString = defaultBookmarks
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
				default:
					return nil, err
				}
			} else {
				bookmarkString = bookmarks.List.String
			}
			tabs := ParseBookmarks(bookmarkString)
			tabStr := r.URL.Query().Get("tab")
			idx, err := strconv.Atoi(tabStr)
			if err != nil || idx < 0 || idx >= len(tabs) {
				idx = 0
			}
			return tabs[idx].Pages, nil
		},
		"bookmarkTabs": func() ([]TabInfo, error) {
			queries := r.Context().Value(ContextValues("queries")).(*Queries)
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			userRef, _ := session.Values["UserRef"].(string)

			bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
			var bookmarkString = defaultBookmarks
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
				default:
					return nil, err
				}
			} else {
				bookmarkString = bookmarks.List.String
			}
			tabsData := ParseBookmarks(bookmarkString)
			var tabs []TabInfo
			for i, t := range tabsData {
				indexName := t.DisplayName()
				if indexName == "" && i == 0 {
					indexName = "Main"
				}
				if indexName != "" {
					href := "/"
					if i != 0 {
						href = fmt.Sprintf("/?tab=%d", i)
					}
					lastSha := ""
					if len(t.Pages) > 0 {
						lastSha = t.Pages[len(t.Pages)-1].Sha()
					}
					tabs = append(tabs, TabInfo{Index: i, Name: t.Name, IndexName: indexName, Href: href, LastPageSha: lastSha})
				}
			}
			return tabs, nil
		},
		"tabName": func() string {
			queries := r.Context().Value(ContextValues("queries")).(*Queries)
			session := r.Context().Value(ContextValues("session")).(*sessions.Session)
			userRef, _ := session.Values["UserRef"].(string)

			bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
			var bookmarkString = defaultBookmarks
			if err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
					return ""
				default:
					return ""
				}
			} else {
				bookmarkString = bookmarks.List.String
			}
			tabs := ParseBookmarks(bookmarkString)
			tabStr := r.URL.Query().Get("tab")
			idx, err := strconv.Atoi(tabStr)
			if err != nil || idx < 0 || idx >= len(tabs) {
				idx = 0
			}
			name := tabs[idx].DisplayName()
			if name == "" && idx == 0 {
				name = "Main"
			}
			return name
		},
		"add1": func(i int) int { return i + 1 },
	}
}
