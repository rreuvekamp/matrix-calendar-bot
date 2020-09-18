package main

func main() {
	initMatrixBot()
	<-make(chan struct{})
}
