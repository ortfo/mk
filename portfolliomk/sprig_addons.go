package main

import "strings"

func IndentWithTabs(tabs int, s string) string {
	pad := strings.Repeat("\t", tabs)
	return pad + strings.Replace(s, "\n", "\n"+pad, -1)
}

func IndentWithTabsNewline(tabs int, s string) string {
	return "\n" + IndentWithTabs(tabs, s)
}
