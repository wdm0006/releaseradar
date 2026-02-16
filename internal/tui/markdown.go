package tui

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

func boolPtr(b bool) *bool    { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint    { return &u }

// appStyleConfig defines a custom glamour style matching the app's dark theme.
var appStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "\n",
			Color:       stringPtr("#F3F4F6"),
		},
		Margin: uintPtr(0),
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:     stringPtr("#8B5CF6"),
			Bold:      boolPtr(true),
			Underline: boolPtr(true),
			Prefix:    "# ",
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr("#F59E0B"),
			Bold:   boolPtr(true),
			Prefix: "## ",
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr("#06B6D4"),
			Bold:   boolPtr(true),
			Prefix: "### ",
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr("#06B6D4"),
			Bold:   boolPtr(true),
			Prefix: "#### ",
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr("#06B6D4"),
			Bold:   boolPtr(true),
			Prefix: "##### ",
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr("#06B6D4"),
			Bold:   boolPtr(true),
			Prefix: "###### ",
		},
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
		Margin:         uintPtr(0),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: stringPtr("#9CA3AF"),
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr("│ "),
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{},
		LevelIndent: 2,
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
		Color:       stringPtr("#F59E0B"),
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
		Color:       stringPtr("#F59E0B"),
	},
	Strong: ansi.StylePrimitive{
		Bold:  boolPtr(true),
		Color: stringPtr("#F3F4F6"),
	},
	Emph: ansi.StylePrimitive{
		Italic: boolPtr(true),
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  stringPtr("#374151"),
		Format: "\n──────────────────────\n",
	},
	Link: ansi.StylePrimitive{
		Color:     stringPtr("#06B6D4"),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: stringPtr("#06B6D4"),
		Bold:  boolPtr(true),
	},
	Image: ansi.StylePrimitive{
		Color:     stringPtr("#06B6D4"),
		Underline: boolPtr(true),
	},
	ImageText: ansi.StylePrimitive{
		Color:  stringPtr("#9CA3AF"),
		Format: "Image: {{.text}}",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr("#06B6D4"),
			BackgroundColor: stringPtr("#1F2937"),
			Prefix:          " ",
			Suffix:          " ",
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("#D1D5DB"),
			},
			Margin: uintPtr(0),
		},
		Chroma: &ansi.Chroma{
			Text: ansi.StylePrimitive{
				Color: stringPtr("#D1D5DB"),
			},
			Comment: ansi.StylePrimitive{
				Color: stringPtr("#6B7280"),
			},
			Keyword: ansi.StylePrimitive{
				Color: stringPtr("#8B5CF6"),
			},
			KeywordReserved: ansi.StylePrimitive{
				Color: stringPtr("#8B5CF6"),
			},
			KeywordType: ansi.StylePrimitive{
				Color: stringPtr("#06B6D4"),
			},
			Operator: ansi.StylePrimitive{
				Color: stringPtr("#F59E0B"),
			},
			Name: ansi.StylePrimitive{
				Color: stringPtr("#F3F4F6"),
			},
			NameFunction: ansi.StylePrimitive{
				Color: stringPtr("#06B6D4"),
			},
			NameClass: ansi.StylePrimitive{
				Color: stringPtr("#F59E0B"),
			},
			LiteralString: ansi.StylePrimitive{
				Color: stringPtr("#34D399"),
			},
			LiteralNumber: ansi.StylePrimitive{
				Color: stringPtr("#F59E0B"),
			},
			GenericInserted: ansi.StylePrimitive{
				Color: stringPtr("#34D399"),
			},
			GenericDeleted: ansi.StylePrimitive{
				Color: stringPtr("#F87171"),
			},
			GenericEmph: ansi.StylePrimitive{
				Italic: boolPtr(true),
			},
			GenericStrong: ansi.StylePrimitive{
				Bold: boolPtr(true),
			},
		},
		Theme: "monokai",
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
		},
		CenterSeparator: stringPtr("┼"),
		ColumnSeparator: stringPtr("│"),
		RowSeparator:    stringPtr("─"),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{},
		Ticked:         "[✓] ",
		Unticked:       "[ ] ",
	},
	DefinitionTerm: ansi.StylePrimitive{
		Bold:  boolPtr(true),
		Color: stringPtr("#F59E0B"),
	},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix: "\n",
	},
}

// newMarkdownRenderer creates a glamour markdown renderer with the app's custom style.
func newMarkdownRenderer(width int) *glamour.TermRenderer {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(appStyleConfig),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	return r
}
