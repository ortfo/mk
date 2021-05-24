package main

import "fmt"

func printfln(text string, a ...interface{}) {
	fmt.Printf(text+"\n", a...)
}

func printerr(explanation string, err error) {
	printfln(explanation+": %s", err)
}
