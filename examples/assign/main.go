package main

import "fmt"

type User struct {
	Name string
}

func main() {
	uv := User{}
	up := &User{}

	fmt.Println(uv, up)
}
