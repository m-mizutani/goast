package main

import "fmt"

type User struct{}

func main() {
	u1 := User{}  // goast.sync: policy/data/assign/value/main.json
	u2 := &User{} // goast.sync: policy/data/assign/ptr/main.json
	var u3 User   // goast.sync: policy/data/assign/def/main.json

	fmt.Println(u1, u2, u3)
}
