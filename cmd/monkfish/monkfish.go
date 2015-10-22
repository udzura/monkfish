package main

import "github.com/udzura/monkfish"

func main() {
	err := monkfish.Run()
	if err != nil {
		panic(err)
	}
}
