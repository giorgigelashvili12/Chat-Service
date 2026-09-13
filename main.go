package main

// import "fmt" // format
import (
	"fmt"
	"context"
	"github.com/giorgigelashvili12/Chat-Service/application"
)

func main() {
	app := application.New()
	err := app.Start(context.TODO())
	if err != nil {
		fmt.Println("Failed to start app: ", err)
	}
}
