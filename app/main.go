// main.go

package main

import "os"

func main() {
	m := Monitor{}
	m.Initialize(
		os.Getenv("APP_DB_USERNAME"),
		os.Getenv("APP_DB_PASSWORD"),
		os.Getenv("APP_DB_NAME"))

	m.Run(":8080")
}
