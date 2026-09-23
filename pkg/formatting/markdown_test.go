package formatting

import "testing"

func TestFormatMarkdownMatchesPyMaxEntities(t *testing.T) {
	clean, elements := FormatMarkdown("# Title\n> Quote\nHello **bold** and [site](https://example.com)")
	if clean != "Title\nQuote\nHello bold and site" {
		t.Fatalf("clean=%q", clean)
	}
	wantTypes := []string{"HEADING", "QUOTE", "STRONG", "LINK"}
	wantFrom := []int{0, 6, 18, 27}
	wantLength := []int{5, 5, 4, 4}
	if len(elements) != len(wantTypes) {
		t.Fatalf("elements=%#v", elements)
	}
	for index := range elements {
		if elements[index].Type != wantTypes[index] || elements[index].From != wantFrom[index] || elements[index].Length != wantLength[index] {
			t.Fatalf("element[%d]=%#v", index, elements[index])
		}
	}
	if elements[3].Attributes == nil || elements[3].Attributes.URL != "https://example.com" {
		t.Fatalf("link attributes=%#v", elements[3].Attributes)
	}
}

func TestFormatMarkdownUsesUTF16Offsets(t *testing.T) {
	clean, elements := FormatMarkdown("😀 **bold**")
	if clean != "😀 bold" || len(elements) != 1 || elements[0].From != 3 || elements[0].Length != 4 {
		t.Fatalf("clean=%q elements=%#v", clean, elements)
	}
}
