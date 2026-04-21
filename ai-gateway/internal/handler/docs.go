package handler

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	scriptRegex  = regexp.MustCompile(`<script[^>]*>.*?</script>`)
	eventRegex  = regexp.MustCompile(`\bon\w+\s*=`)
	styleRegex  = regexp.MustCompile(`expression\s*\(`)
)

func sanitizeHTML(html string) string {
	html = scriptRegex.ReplaceAllString(html, "")
	html = eventRegex.ReplaceAllString(html, "")
	html = styleRegex.ReplaceAllString(html, "")
	return html
}

func extractTextContent(html string) string {
	textRegex := regexp.MustCompile(`<[^>]+>`)
	return textRegex.ReplaceAllString(html, " ")
}

type DocsHandler struct {
	docsPath string
}

func NewDocsHandler() *DocsHandler {
	return &DocsHandler{
		docsPath: "./docs",
	}
}

func (h *DocsHandler) Render(c *gin.Context) {
	docPath := c.Query("path")
	if docPath == "" {
		docPath = "/docs/README.md"
	}
	
	docPath = strings.TrimPrefix(docPath, "/")
	if !strings.HasSuffix(docPath, ".md") {
		docPath += ".md"
	}
	
	fullPath := filepath.Join(h.docsPath, docPath)
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}
	
	content := string(data)
	
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (h *DocsHandler) RenderHTML(c *gin.Context) {
	docPath := c.Query("path")
	if docPath == "" {
		docPath = "/docs/README.md"
	}
	
	docPath = strings.TrimPrefix(docPath, "/")
	if !strings.HasSuffix(docPath, ".md") {
		docPath += ".md"
	}
	
	fullPath := filepath.Join(h.docsPath, docPath)
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}
	
	content := string(data)
	
	html := markdownToHTML(content)
	html = sanitizeHTML(html)
	
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func markdownToHTML(md string) string {
	var buf bytes.Buffer
	
	lines := strings.Split(md, "\n")
	inCodeBlock := false
	
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				buf.WriteString("<pre><code>")
				inCodeBlock = true
			} else {
				buf.WriteString("</code></pre>")
				inCodeBlock = false
			}
			continue
		}
		
		if inCodeBlock {
			buf.WriteString(escapeHTML(line))
			buf.WriteString("\n")
			continue
		}
		
		switch {
		case strings.HasPrefix(line, "# "):
			buf.WriteString("<h1>")
			buf.WriteString(escapeHTML(strings.TrimPrefix(line, "# ")))
			buf.WriteString("</h1>\n")
		case strings.HasPrefix(line, "## "):
			buf.WriteString("<h2>")
			buf.WriteString(escapeHTML(strings.TrimPrefix(line, "## ")))
			buf.WriteString("</h2>\n")
		case strings.HasPrefix(line, "### "):
			buf.WriteString("<h3>")
			buf.WriteString(escapeHTML(strings.TrimPrefix(line, "### ")))
			buf.WriteString("</h3>\n")
		case strings.HasPrefix(line, "- "):
			buf.WriteString("<li>")
			buf.WriteString(escapeHTML(strings.TrimPrefix(line, "- ")))
			buf.WriteString("</li>\n")
		case strings.HasPrefix(line, "| "):
			row := strings.Trim(line, "| ")
			cells := strings.Split(row, "|")
			buf.WriteString("<tr>")
			for _, cell := range cells {
				buf.WriteString("<td>")
				buf.WriteString(escapeHTML(strings.TrimSpace(cell)))
				buf.WriteString("</td>")
			}
			buf.WriteString("</tr>\n")
		case strings.TrimSpace(line) == "---":
			buf.WriteString("<hr>\n")
		case strings.TrimSpace(line) == "":
			buf.WriteString("<br>\n")
		default:
			formatted := formatInlineMarkdown(line)
			buf.WriteString("<p>")
			buf.WriteString(formatted)
			buf.WriteString("</p>\n")
		}
	}
	
	return buf.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

func formatInlineMarkdown(s string) string {
	s = escapeHTML(s)
	
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	s = linkRegex.ReplaceAllString(s, `<a href="$2">$1</a>`)
	
	codeRegex := regexp.MustCompile("`([^`]+)`")
	s = codeRegex.ReplaceAllString(s, "<code>$1</code>")
	
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	s = boldRegex.ReplaceAllString(s, "<strong>$1</strong>")
	
	italicRegex := regexp.MustCompile(`\*([^*]+)\*`)
	s = italicRegex.ReplaceAllString(s, "<em>$1</em>")
	
	return s
}
