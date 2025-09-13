package style

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var DetailedItem any

var Base = lipgloss.NewStyle()

var ColorYellow = lipgloss.Color("#e9a015")
var ColorRed = lipgloss.Color("#d0523c")
var ColorGreen = lipgloss.Color("#00aa00")
var ColorDarkRed = lipgloss.Color("#aa0000")
var ColorGray = lipgloss.Color("#888888")
var ColorBlack = lipgloss.Color("#111111")

var H1Style = lipgloss.NewStyle().Background(ColorYellow).Foreground(ColorBlack).Bold(true)
var H2Style = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
var BlockStyle = lipgloss.NewStyle().MarginLeft(4)
var EmphStyle = lipgloss.NewStyle().Foreground(ColorRed).Italic(true)
var SubtitleStyle = lipgloss.NewStyle().Foreground(ColorGray).Italic(true)
var SuccessStyle = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
var ErrorStyle = lipgloss.NewStyle().Foreground(ColorDarkRed).Bold(true)
var UnknownStyle = lipgloss.NewStyle().Foreground(ColorGray).Bold(true)
var IdStyle = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)

func Title(in string) string {
	in = fmt.Sprintf(" %s ", in)
	return fmt.Sprintf("\n%s\n", H1Style.Render(in))
}

func BlockTitle(in string) string {
	in = fmt.Sprintf("→ %s:", in)
	return fmt.Sprintf("\n%s\n", H2Style.Render(in))
}

func InlineBlockTitle(in string) string {
	in = fmt.Sprintf("→ %s:", in)
	return fmt.Sprintf("\n%s ", H2Style.Render(in))
}

func Block(in string) string {
	return BlockStyle.Render(strings.Trim(in, "\n"))
}

func Emph(in string) string {
	return EmphStyle.Render(in)
}

func SpacedBlock(in string) string {
	in = strings.Trim(in, "\n")
	if in == "" {
		return "\n"
	}
	return fmt.Sprintf("\n%s\n", in)
}

func Item(in string) string {
	return fmt.Sprintf(" • %s\n", in)
}

func SubItem(in string) string {
	return fmt.Sprintf("   • %s\n", in)
}

func Subtitle(in string) string {
	return fmt.Sprintf("\n%s\n", SubtitleStyle.Render(in))
}

func RenderSuccess(in string) string {
	return SuccessStyle.Render(in)
}

func RenderError(in string) string {
	return ErrorStyle.Render(in)
}

func RenderUnknown(in string) string {
	return UnknownStyle.Render(in)
}

func RenderID(in string) string {
	return IdStyle.Render(in)
}
