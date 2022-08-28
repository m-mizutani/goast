package main

import "fmt"

type exampleType struct {
	FieldString string
	FieldInt    int
}

const exampleConst = "blue"

// exampleVar is orange
var exampleVar = "orange"

func main() {
	fmt.Println("test:", exampleConst, exampleVar)

	examples := make([]*exampleType, 5)

	oneExample := &exampleType{
		FieldString: "blue",
		FieldInt:    5,
	}

	for _, e := range examples {
		e.FieldInt = oneExample.FieldInt
	}

	MyTestFunc(oneExample)
}

// MyTestFunc is test
func MyTestFunc(obj *exampleType) bool {
	return obj.FieldInt == 0
}
