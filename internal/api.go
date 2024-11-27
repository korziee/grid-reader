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

func (s *SudokuServer) pong(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("pong\n"))
}

func (s *SudokuServer) readGrid(w http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(5 << 20) // 5MB
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

	grid := GridFromImage(img, header.Filename)
	if err := grid.SplitCells(ModeComparison); err != nil {
		http.Error(w, "failed to split cells", http.StatusInternalServerError)
		return
	}

	if err := grid.Process(s.worker.jobs); err != nil {
		http.Error(w, "failed to process cells", http.StatusInternalServerError)
		return
	}

	type CellRes struct {
		Identifier   string   `json:"identifier"`
		Type         CellType `json:"type"`
		Val          int      `json:"val"`
		Placeholders []int    `json:"placeholders"`
	}

	type Res struct {
		ID                      string      `json:"id"`
		CharacterRepresentation string      `json:"character_representation"`
		GridRepresentation      [][]CellRes `json:"grid_json"`
	}

	gridRep := make([][]CellRes, len(grid.Cells))

	for rIdx, row := range grid.Cells {
		gridRep[rIdx] = make([]CellRes, len(row))
		for cIdx, cell := range row {
			gridRep[rIdx][cIdx] = CellRes{
				Identifier:   cell.Identifier,
				Type:         cell.Type(),
				Val:          cell.comparisonValue,
				Placeholders: cell.comparisonPlaceholders,
			}
		}
	}

	r := Res{
		ID:                      header.Filename,
		CharacterRepresentation: grid.String(),
		GridRepresentation:      gridRep,
	}

	resB, err := json.Marshal(r)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(resB)
}

func (s *SudokuServer) Start() {
	s.worker.Start()
	http.HandleFunc("/ping", s.pong)
	http.HandleFunc("/read-grid", s.readGrid)

	fmt.Println("listening on port 8080")
	// todo: make this localhost when not in container
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func NewSudokuServer() *SudokuServer {
	return &SudokuServer{
		worker: NewGridWorker(),
	}
}
