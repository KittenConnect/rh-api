package util

import (
	"fmt"
	"github.com/fatih/color"
	"os"
)

func Color(c color.Attribute, prefix string, s string, v ...any) {
	colored := color.New(c).SprintFunc()
	fmt.Printf(fmt.Sprintf(colored("[%s] %s\n"), prefix, s), v...)
}

func Err(s string, v ...any) {
	// red := color.New(color.FgRed).SprintFunc()
	// fmt.Printf(fmt.Sprintf(red("%s %s\n"), "[ERROR]", s), v...)
	Color(color.FgRed, "ERROR", s, v...)
	os.Exit(-1)
}

func Warn(s string, v ...any) {
	// orange := color.New(color.FgYellow).SprintFunc()
	// fmt.Printf("%s %s\n", orange("[WARN]"), orange(t))
	Color(color.FgYellow, "WARN", s, v...)
}

func Info(s string, v ...any) {
	// blue := color.New(color.FgBlue).SprintFunc()
	// fmt.Printf("%s %s\n", blue("[INFO]"), blue(t))
	Color(color.FgBlue, "INFO", s, v...)
}

func Success(s string, v ...any) {
	// green := color.New(color.FgGreen).SprintFunc()
	// fmt.Printf("%s %s\n", green("[SUCCESS]"), green(t))
	Color(color.FgGreen, "SUCCESS", s, v...)
}
