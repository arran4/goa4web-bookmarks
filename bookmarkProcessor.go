package a4webbm

import (
	"strings"
)

type BookmarkEntry struct {
	Url  string
	Name string
}

type BookmarkCategory struct {
	Name    string
	Entries []*BookmarkEntry
}

type BookmarkColumn struct {
	Categories []*BookmarkCategory
}

func PreprocessBookmarks(bookmarks string) []*BookmarkColumn {
	lines := strings.Split(bookmarks, "\n")
	var result = []*BookmarkColumn{{}}
	var currentCategory *BookmarkCategory

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.EqualFold(line, "column") {
			if currentCategory != nil {
				result[len(result)-1].Categories = append(result[len(result)-1].Categories, currentCategory)
				currentCategory = nil
			}
			result = append(result, &BookmarkColumn{})
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) > 0 && strings.EqualFold(parts[0], "Category:") {
			categoryName := strings.Join(parts[1:], " ")
			if currentCategory == nil {
				currentCategory = &BookmarkCategory{Name: categoryName}
			} else if currentCategory.Name != "" {
				result[len(result)-1].Categories = append(result[len(result)-1].Categories, currentCategory)
				currentCategory = &BookmarkCategory{Name: categoryName}
			} else {
				currentCategory.Name = categoryName
			}
		} else if len(parts) > 0 && currentCategory != nil {
			var entry BookmarkEntry
			entry.Url = parts[0]
			entry.Name = parts[0]
			if len(parts) > 1 {
				entry.Name = strings.Join(parts[1:], " ")
			}
			currentCategory.Entries = append(currentCategory.Entries, &entry)
		}
	}

	if currentCategory != nil && currentCategory.Name != "" {
		result[len(result)-1].Categories = append(result[len(result)-1].Categories, currentCategory)
	}

	return result
}
