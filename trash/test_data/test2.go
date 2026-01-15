//go:build ignore

package main

func newFunction() {
	return "newFunction called"
}

func anotherOldFunction() {
	result := newFunction()
	return result
}
