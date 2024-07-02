package util

import (
	"fmt"
	"github.com/fatih/color"
	"os"
)

func Err(t string) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s %s\n", red("[ERROR]"), red(t))
	os.Exit(-1)
}

func Warn(t string) {
	orange := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s %s\n", orange("[WARN]"), orange(t))
}

func Info(t string) {
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("%s %s\n", blue("[INFO]"), blue(t))
}

func Success(t string) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s %s\n", green("[SUCCESS]"), green(t))
}
