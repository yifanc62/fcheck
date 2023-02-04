package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var (
	successColorFunc = color.New(color.Bold, color.FgWhite, color.BgGreen).PrintfFunc()
	failColorFunc    = color.New(color.Bold, color.FgWhite, color.BgYellow).PrintfFunc()
)

func PrintStatus(isSuccess bool, label, path string) {
	colorFunc := failColorFunc
	if isSuccess {
		colorFunc = successColorFunc
	}
	colorFunc("[%s]", label)
	fmt.Println(strings.Repeat(" ", 10-len(label)), path)
}
