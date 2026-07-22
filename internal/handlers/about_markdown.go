package handlers

import (
	"html"
	"html/template"
	"regexp"
	"strings"
)

var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

type aboutContent struct {
	HTML template.HTML
}

func renderAboutMarkdown(markdown string) aboutContent {
	var builder strings.Builder
	inList := false

	for _, rawLine := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			if inList {
				builder.WriteString("</ul>")
				inList = false
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, "### "):
			if inList {
				builder.WriteString("</ul>")
				inList = false
			}
			builder.WriteString("<h3>")
			builder.WriteString(renderInlineMarkdown(strings.TrimPrefix(line, "### ")))
			builder.WriteString("</h3>")
		case strings.HasPrefix(line, "## "):
			if inList {
				builder.WriteString("</ul>")
				inList = false
			}
			builder.WriteString("<h2>")
			builder.WriteString(renderInlineMarkdown(strings.TrimPrefix(line, "## ")))
			builder.WriteString("</h2>")
		case strings.HasPrefix(line, "# "):
			if inList {
				builder.WriteString("</ul>")
				inList = false
			}
			builder.WriteString("<h1>")
			builder.WriteString(renderInlineMarkdown(strings.TrimPrefix(line, "# ")))
			builder.WriteString("</h1>")
		case strings.HasPrefix(line, "- "):
			if !inList {
				builder.WriteString("<ul>")
				inList = true
			}
			builder.WriteString("<li>")
			builder.WriteString(renderInlineMarkdown(strings.TrimPrefix(line, "- ")))
			builder.WriteString("</li>")
		default:
			if inList {
				builder.WriteString("</ul>")
				inList = false
			}
			builder.WriteString("<p>")
			builder.WriteString(renderInlineMarkdown(line))
			builder.WriteString("</p>")
		}
	}

	if inList {
		builder.WriteString("</ul>")
	}

	return aboutContent{HTML: template.HTML(builder.String())}
}

func renderInlineMarkdown(value string) string {
	escaped := html.EscapeString(value)
	return markdownLinkPattern.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := markdownLinkPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		label := parts[1]
		url := parts[2]
		if !safeAboutURL(url) {
			return label
		}
		return `<a href="` + html.EscapeString(url) + `" rel="noopener noreferrer" target="_blank">` + label + `</a>`
	})
}

func safeAboutURL(value string) bool {
	return strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "mailto:")
}
