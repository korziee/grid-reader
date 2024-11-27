package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/korziee/spike-sudoku-parse/internal"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error loading .env file")
	}

	internal.LoadLogger()
	server := internal.NewSudokuServer()
	server.Start()
}
