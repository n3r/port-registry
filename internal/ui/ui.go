package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// ANSI color numbers (0-15) to respect terminal themes.
var (
	colorGreen  = lipgloss.ANSIColor(2)
	colorRed    = lipgloss.ANSIColor(1)
	colorYellow = lipgloss.ANSIColor(3)
	colorCyan   = lipgloss.ANSIColor(6)
	colorGray   = lipgloss.ANSIColor(8)
)

// Styles for direct use.
var (
	StyleSuccess = lipgloss.NewStyle().Foreground(colorGreen)
	StyleError   = lipgloss.NewStyle().Foreground(colorRed)
	StyleWarning = lipgloss.NewStyle().Foreground(colorYellow)
	StyleInfo    = lipgloss.NewStyle().Foreground(colorCyan)
	StyleSubtle  = lipgloss.NewStyle().Foreground(colorGray)
	StyleBold    = lipgloss.NewStyle().Bold(true)
)

// Symbols for status indicators.
const (
	SymCheck   = "✓"
	SymCross   = "✗"
	SymArrow   = "→"
	SymBullet  = "•"
	SymWarning = "⚠"
)

// Success returns a green check-prefixed message.
func Success(msg string) string {
	return StyleSuccess.Render(SymCheck + " " + msg)
}

// Successf returns a green check-prefixed formatted message.
func Successf(format string, a ...any) string {
	return Success(fmt.Sprintf(format, a...))
}

// Error returns a red cross-prefixed message.
func Error(msg string) string {
	return StyleError.Render(SymCross + " " + msg)
}

// Errorf returns a red cross-prefixed formatted message.
func Errorf(format string, a ...any) string {
	return Error(fmt.Sprintf(format, a...))
}

// Warning returns a yellow warning-prefixed message.
func Warning(msg string) string {
	return StyleWarning.Render(SymWarning + " " + msg)
}

// Warningf returns a yellow warning-prefixed formatted message.
func Warningf(format string, a ...any) string {
	return Warning(fmt.Sprintf(format, a...))
}

// Info returns a cyan bullet-prefixed message.
func Info(msg string) string {
	return StyleInfo.Render(SymBullet + " " + msg)
}

// Infof returns a cyan bullet-prefixed formatted message.
func Infof(format string, a ...any) string {
	return Info(fmt.Sprintf(format, a...))
}

// Subtle returns dim gray text.
func Subtle(msg string) string {
	return StyleSubtle.Render(msg)
}

// Bold returns bold text.
func Bold(msg string) string {
	return StyleBold.Render(msg)
}

// Table renders a styled table with rounded borders.
func Table(headers []string, rows [][]string) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorGray)).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
			}
			return lipgloss.NewStyle()
		})
	return t.Render()
}

// UsageTitle returns a bold title for usage text.
func UsageTitle(text string) string {
	return StyleBold.Render(text)
}

// UsageHeader returns a bold cyan section header.
func UsageHeader(text string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(colorCyan).Render(text)
}

// UsageCommand formats a command name and description, aligned.
func UsageCommand(name, desc string) string {
	styled := StyleSuccess.Render(fmt.Sprintf("  %-10s", name))
	return styled + " " + desc
}
