package main

import (
	"fmt"
)

func main() {

	a := make([]int, 0)

	a = append(a, 4)

	a = append(a, 8)

	fmt.Println(len(a))

	for _, value := range a {
		fmt.Println(value)
	}

	thingy := "shit"
	thingy = thingy + "shoot"
	fmt.Println(thingy)

}
