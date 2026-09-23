// Package formatting converts the Max-supported Markdown subset to protocol
// text elements. Element offsets and lengths are UTF-16 code units.
package formatting

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

var markers = []struct{ marker, kind string }{
	{"```", "CODE"}, {"**", "STRONG"}, {"__", "UNDERLINE"},
	{"~~", "STRIKETHROUGH"}, {"`", "MONOSPACED"}, {"_", "EMPHASIZED"}, {"*", "EMPHASIZED"},
}

func utf16Len(value string) int { return len(utf16.Encode([]rune(value))) }

// FormatMarkdown removes supported Markdown markers and returns Max elements.
func FormatMarkdown(text string) (string, []types.Element) {
	var clean strings.Builder
	elements := make([]types.Element, 0)
	active := make(map[string]int)
	position := 0
	lineStart := true

	for i := 0; i < len(text); {
		if text[i] == '[' {
			if labelEnd := strings.IndexByte(text[i+1:], ']'); labelEnd >= 0 {
				labelEnd += i + 1
				if labelEnd+1 < len(text) && text[labelEnd+1] == '(' {
					if urlEnd := strings.IndexByte(text[labelEnd+2:], ')'); urlEnd >= 0 {
						urlEnd += labelEnd + 2
						label, target := text[i+1:labelEnd], text[labelEnd+2:urlEnd]
						if label != "" && target != "" {
							length := utf16Len(label)
							clean.WriteString(label)
							elements = append(elements, types.Element{Type: "LINK", From: position, Length: length, Attributes: &types.ElementAttributes{URL: target}})
							position += length
							i = urlEnd + 1
							lineStart = false
							continue
						}
					}
				}
			}
		}

		if lineStart && text[i] == '#' {
			j := i
			for j < len(text) && text[j] == '#' {
				j++
			}
			if j < len(text) && text[j] == ' ' {
				j++
				end := strings.IndexByte(text[j:], '\n')
				if end < 0 {
					end = len(text) - j
				}
				value := text[j : j+end]
				clean.WriteString(value)
				length := utf16Len(value)
				if length > 0 {
					elements = append(elements, types.Element{Type: "HEADING", From: position, Length: length})
				}
				position += length
				i = j + end
				lineStart = false
				continue
			}
		}
		if lineStart && text[i] == '>' {
			j := i + 1
			if j < len(text) && text[j] == ' ' {
				j++
			}
			end := strings.IndexByte(text[j:], '\n')
			if end < 0 {
				end = len(text) - j
			}
			value := text[j : j+end]
			clean.WriteString(value)
			length := utf16Len(value)
			if length > 0 {
				elements = append(elements, types.Element{Type: "QUOTE", From: position, Length: length})
			}
			position += length
			i = j + end
			lineStart = false
			continue
		}

		handled := false
		for _, item := range markers {
			if !strings.HasPrefix(text[i:], item.marker) {
				continue
			}
			if start, ok := active[item.marker]; ok {
				if length := position - start; length > 0 {
					elements = append(elements, types.Element{Type: item.kind, From: start, Length: length})
				}
				delete(active, item.marker)
				i += len(item.marker)
				handled = true
				break
			}
			remainder := text[i+len(item.marker):]
			lineEnd := strings.IndexByte(remainder, '\n')
			closing := strings.Index(remainder, item.marker)
			if closing <= 0 || (item.marker != "```" && lineEnd >= 0 && closing > lineEnd) {
				clean.WriteString(item.marker)
				position += utf16Len(item.marker)
				i += len(item.marker)
				handled = true
				break
			}
			active[item.marker] = position
			i += len(item.marker)
			if item.marker == "```" {
				if newline := strings.IndexByte(text[i:i+closing], '\n'); newline >= 0 {
					i += newline + 1
				}
			}
			handled = true
			break
		}
		if handled {
			lineStart = false
			continue
		}

		r, size := utf8.DecodeRuneInString(text[i:])
		clean.WriteString(text[i : i+size])
		position += utf16Len(string(r))
		lineStart = r == '\n'
		i += size
	}
	return clean.String(), elements
}
