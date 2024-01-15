package main

import "fmt"

// Common structure with shared methods
type SharedMethods struct {
	// Fields specific to SharedMethods can also be declared here
}

// Method shared by both structures
func (s *SharedMethods) SharedMethod() {
	fmt.Println("This method is shared 1.")
}

// Structure 1
type Struct1 struct {
	SharedMethods // Embedding SharedMethods into Struct1
	// Additional fields specific to Struct1 can be declared here
}

// Structure 2
type Struct2 struct {
	SharedMethods // Embedding SharedMethods into Struct2
	// Additional fields specific to Struct2 can be declared here
}

func (s *Struct2) doStruct2() {

}

func (s *Struct2) SharedMethod() {
	fmt.Println("This method is shared 2.")
}

func acceptsSm(v SharedMethods) {

}

func main() {
	// Create instances of Struct1 and Struct2
	instance1 := Struct1{}
	instance2 := Struct2{}

	// Call the shared method on both instances
	instance1.SharedMethod()
	instance2.SharedMethod()
	instance2.doStruct2()

}
