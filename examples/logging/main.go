package main

import "fmt"

func logging(msg string) {
	fmt.Println("log:", msg)
}

func main() {
	f1()
	f2()
}

func f1() {
	logging("hello")
}

func f2() {
	// do not call logging
}
