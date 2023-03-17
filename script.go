package main

import (
	_ "flag"
	"fmt"
	_ "log"
	_ "net/http"

	_ "google.golang.org/api/googleapi/transport"
	_ "google.golang.org/api/youtube/v3"
)

func main() {
	fmt.Println("Hello, world.")
}
