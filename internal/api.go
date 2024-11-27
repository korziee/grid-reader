package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"

	_ "image/png"
)

type SudokuServer struct {
	worker *GridWorker
}

func (s *SudokuServer) handlePing(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("pong\n"))
}

func (s *SudokuServer) handleGrid(w http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(5 << 20) // 5MB max
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		http.Error(w, "failed to get file from multipart form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		http.Error(w, "unable to read file", http.StatusInternalServerError)
		return
	}

	img, _, err := image.Decode(&buf)
	if err != nil {
		http.Error(w, "unable to decode provided image", http.StatusInternalServerError)
		return
	}

	grid := GridFromImage(img)
	if err := grid.SplitCells(ModeComparison); err != nil {
		http.Error(w, "failed to split cells", http.StatusInternalServerError)
		return
	}

	if err := grid.Process(s.worker.jobs); err != nil {
		http.Error(w, "failed to process cells", http.StatusInternalServerError)
		return
	}

	type Res struct {
		ID                      string `json:"id"`
		CharacterRepresentation string `json:"character_representation"`
	}

	resB, err := json.Marshal(Res{ID: header.Filename, CharacterRepresentation: grid.String()})
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	// todo: run strace
	w.Header().Add("Content-Type", "application/json")
	w.Write(resB)
}

func (s *SudokuServer) Start() {
	s.worker.Start()
	http.HandleFunc("/ping", s.handlePing)
	http.HandleFunc("/grid", s.handleGrid)

	fmt.Println("listening on port 8080")
	http.ListenAndServe("localhost:8080", nil)
}

func NewSudokuServer() *SudokuServer {
	return &SudokuServer{
		worker: NewGridWorker(),
	}
}
