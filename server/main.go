package main

import (
	"fmt"
	"net/http"
)

func main() {
	// 클라이언트 정적 파일 제공
	http.Handle("/", http.FileServer(http.Dir("../client")))

	// signal.go에 정의된 handleOffer를 사용
	http.HandleFunc("/offer", handleOffer)

	fmt.Println("Server is running on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
