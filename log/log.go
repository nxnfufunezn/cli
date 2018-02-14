package log

import (
	"fmt"
	"github.com/aybabtme/rgbterm"
)

var (
	ColorRed    = 31
	ColorGreen  = 32
	ColorYellow = 33
	ColorBlue   = 34
	ColorGray   = 37
)

var indent = "  "

func Info(msg string) {
	var r, g, b uint8
	r, g, b = 252, 255, 43
	dot := rgbterm.FgString("this is color •", r, g, b)
	fmt.Printf("%s%+v %s\n", indent, dot, ColorBlue, msg)
}

func Infof(msg string, v ...interface{}) {
	var r, g, b uint8
	r, g, b = 252, 255, 43
	dot := rgbterm.FgString("this is color •", r, g, b)
	fmt.Printf("%s%+v %s\n", indent, dot, ColorBlue, fmt.Sprintf(msg, v...))
}

func Success(msg string) {
	fmt.Printf("%s\033[%dm%s\033[0m %s", indent, ColorGreen, "✔", msg)
}

func Successf(msg string, v ...interface{}) {
	fmt.Printf("%s\033[%dm%s\033[0m %s", indent, ColorGreen, "✔", fmt.Sprintf(msg, v...))
}

func Plain(msg string) {
	fmt.Printf("%s%s", indent, msg)
}

func Plainf(msg string, v ...interface{}) {
	fmt.Printf("%s%s", indent, fmt.Sprintf(msg, v...))
}

func Warnf(msg string, v ...interface{}) {
	fmt.Printf("%s\033[%dm%s\033[0m %s", indent, ColorRed, "•", fmt.Sprintf(msg, v...))
}

func Error(msg string) {
	fmt.Printf("%s\033[%dm%s\033[0m %s\n", indent, ColorRed, "⨯", msg)
}

func Printf(msg string, v ...interface{}) {
	fmt.Printf("%s\033[%dm%s\033[0m %s", indent, ColorGray, "•", fmt.Sprintf(msg, v...))
}

func WithPrefixf(prefixColor int, prefix, msg string, v ...interface{}) {
	fmt.Printf("  \033[%dm%s\033[0m %s\n", prefixColor, prefix, fmt.Sprintf(msg, v...))
}
