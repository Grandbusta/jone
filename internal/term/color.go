// Package term provides terminal color utilities using ANSI escape codes.
package term

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

// ColorFunc returns a function that wraps text in the given color.
func ColorFunc(color string) func(string) string {
	return func(s string) string {
		return color + s + Reset
	}
}

// Helper functions for common colors
func RedText(s string) string    { return Red + s + Reset }
func GreenText(s string) string  { return Green + s + Reset }
func YellowText(s string) string { return Yellow + s + Reset }
func BlueText(s string) string   { return Blue + s + Reset }
func CyanText(s string) string   { return Cyan + s + Reset }
func BoldText(s string) string   { return Bold + s + Reset }
