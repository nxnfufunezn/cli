package log

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	ColorRed    = color.New(color.FgRed)
	ColorGreen  = color.New(color.FgGreen)
	ColorYellow = color.New(color.FgYellow)
	ColorBlue   = color.New(color.FgBlue)
	ColorGray   = color.New(color.FgHiBlack)
)

var indent = "  "

func Info(msg string) {
	fmt.Fprintf(color.Output, "%s %s %s\n", indent, ColorBlue.Sprint("•"), msg)
}

func Infof(msg string, v ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s %s", indent, ColorBlue.Sprint("•"), fmt.Sprintf(msg, v...))
}

func Success(msg string) {
	fmt.Fprintf(color.Output, "%s %s %s", indent, ColorGreen.Sprint("✔"), msg)
}

func Successf(msg string, v ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s %s", indent, ColorGreen.Sprint("✔"), fmt.Sprintf(msg, v...))
}

func Plain(msg string) {
	fmt.Printf("%s%s", indent, msg)
}

func Plainf(msg string, v ...interface{}) {
	fmt.Printf("%s%s", indent, fmt.Sprintf(msg, v...))
}

func Warnf(msg string, v ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s %s", indent, ColorRed.Sprint("•"), fmt.Sprintf(msg, v...))
}

func Error(msg string) {
	fmt.Fprintf(color.Output, "%s %s %s\n", indent, ColorRed.Sprint("⨯"), msg)
}

func Printf(msg string, v ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s %s", indent, ColorGray.Sprint("•"), fmt.Sprintf(msg, v...))
}
