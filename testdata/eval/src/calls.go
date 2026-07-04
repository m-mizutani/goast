package sample

import "fmt"

func greet(name string) {
	fmt.Println("hello", name)
	fmt.Printf("hi %s\n", name)
	print("debug")
}

func report(values []int) string {
	out := fmt.Sprintf("%d values", len(values))
	for _, v := range values {
		if v > 10 {
			fmt.Println("big value", v)
		}
	}
	return out
}
