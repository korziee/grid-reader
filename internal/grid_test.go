package internal

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/gographics/imagick.v3/imagick"
)

func newTestGrid(rows [][]string, mode Mode) *Grid {
	g := &Grid{
		Cells: [9][9]*Cell{},
	}

	for i, row := range rows {
		g.Cells[i] = [9]*Cell{}
		for j, val := range row {
			c := &Cell{
				Identifier:             fmt.Sprintf("R%dC%d", i+1, j+1),
				comparisonValue:        -1,
				comparisonPlaceholders: []int{},
				ocrValue:               -1,
				ocrPlaceholders:        []int{},
				mode:                   mode,
			}
			if val == "" {
				g.Cells[i][j] = c
				continue
			}

			if val[0] == 'p' {
				for _, v := range val[1:] {
					i, err := strconv.Atoi(string(v))
					if err != nil {
						panic(err)
					}
					c.ocrPlaceholders = append(c.ocrPlaceholders, i)
					c.comparisonPlaceholders = append(c.comparisonPlaceholders, i)
				}
			}

			if len(val) == 1 {
				i, err := strconv.Atoi(val)
				if err != nil {
					panic(err)
				}
				c.ocrValue = i
				c.comparisonValue = i
			}

			g.Cells[i][j] = c
		}
	}
	return g
}

func TestGrid_Process_NYT_OCR(t *testing.T) {
	os.Setenv("DEBUG", "false")
	imagick.Initialize()
	defer imagick.Terminate()

	file, err := os.Open("../nyt.png")
	if err != nil {
		fmt.Println("Error opening image file:", err)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("Error decoding image:", err)
		return
	}

	g := GridFromImage(img, "TestGrid_Process_NYT_OCR")
	err = g.SplitCells(ModeOCR)
	if err != nil {
		t.Error(err)
	}

	testGrid := newTestGrid([][]string{
		{"p123479", "3", "p12", "", "1", "", "2", "", "6"},
		{"7", "p1", "5", "2", "6", "9", "", "", ""},
		{"p89", "p159", "p45", "3", "8", "", "7", "4", ""},
		{"5", "", "", "", "", "", "", "8", "2"},
		{"2", "1", "", "4", "", "3", "", "", ""},
		{"", "6", "7", "5", "", "", "", "", ""},
		{"", "", "9", "1", "", "", "4", "5", ""},
		{"", "", "1", "7", "", "4", "9", "", ""},
		{"", "", "", "", "", "", "", "", ""},
	}, ModeOCR)

	t.Run("digits", func(tt *testing.T) {
		for rowIdx, row := range g.Cells {
			for cellIdx, cell := range row {
				truth := testGrid.Cells[rowIdx][cellIdx]
				if truth.Type() != CellTypeValue {
					continue
				}

				t.Run(fmt.Sprintf("cell_%s", cell.Identifier), func(ttt *testing.T) {
					assert.Equal(ttt, CellTypeValue, cell.Type())
					assert.Equal(ttt, truth.ocrValue, cell.ocrValue)
				})
			}
		}
	})

	t.Run("placeholders", func(tt *testing.T) {
		for rowIdx, row := range g.Cells {
			for cellIdx, cell := range row {
				truth := testGrid.Cells[rowIdx][cellIdx]
				if truth.Type() != CellTypePlaceholders {
					continue
				}

				t.Run(fmt.Sprintf("cell_%s", cell.Identifier), func(ttt *testing.T) {
					assert.Equal(ttt, CellTypePlaceholders, cell.Type())
					assert.Equal(ttt, truth.ocrPlaceholders, cell.ocrPlaceholders)
				})
			}
		}
	})

	t.Run("entire grid", func(tt *testing.T) {
		for rowIdx, row := range g.Cells {
			for cellIdx, cell := range row {
				truth := testGrid.Cells[rowIdx][cellIdx]

				t.Run(fmt.Sprintf("cell_%s", cell.Identifier), func(ttt *testing.T) {
					switch expression := truth.Type(); expression {
					case CellTypeEmpty:
						assert.Equal(tt, CellTypeEmpty, cell.Type())
					case CellTypeValue:
						assert.Equal(tt, CellTypeValue, cell.Type())
						assert.Equal(tt, truth.ocrValue, cell.ocrValue)
					case CellTypePlaceholders:
						assert.Equal(tt, CellTypePlaceholders, cell.Type())
						assert.Equal(tt, truth.ocrPlaceholders, cell.ocrPlaceholders)
					}
				})
			}
		}
	})
}

func TestGrid_Process_NYT_Comparison(t *testing.T) {
	os.Setenv("DEBUG", "false")
	imagick.Initialize()
	defer imagick.Terminate()

	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	gridsPath := path.Join(currentDir, "../grids")
	entries, err := os.ReadDir(gridsPath)
	if err != nil {
		panic(err)
	}

	for idx, e := range entries {
		gridFile, err := os.Open(path.Join(gridsPath, e.Name(), "grid.png"))
		if err != nil {
			panic(fmt.Errorf("opening image file: %v", err))
		}
		defer gridFile.Close()

		img, _, err := image.Decode(gridFile)
		if err != nil {
			panic(fmt.Errorf("decoding image: %v", err))
		}

		truthTableFile, err := os.Open(path.Join(gridsPath, e.Name(), "truth.json"))
		if err != nil {
			panic(fmt.Errorf("opening truth table file: %v", err))
		}
		defer truthTableFile.Close()

		ttBytes, err := io.ReadAll(truthTableFile)
		if err != nil {
			panic(fmt.Errorf("reading truth table file: %v", err))
		}

		var truthTable [][]string
		json.Unmarshal(ttBytes, &truthTable)

		truthTableGrid := newTestGrid(truthTable, ModeComparison)

		t.Run(fmt.Sprintf("grid_%d", idx+1), func(tt *testing.T) {

			tt.Run("digits", func(ttt *testing.T) {
				g := GridFromImage(img, fmt.Sprintf("TestGrid_Process_NYT_Comparison_%d", idx+1))
				if err := g.SplitCells(ModeComparison); err != nil {
					t.Error(err)
				}
				for rowIdx, row := range g.Cells {
					for cellIdx, cell := range row {
						truth := truthTableGrid.Cells[rowIdx][cellIdx]
						if truth.Type() != CellTypeValue {
							continue
						}

						ttt.Run(fmt.Sprintf("cell_%s", cell.Identifier), func(tttt *testing.T) {
							if err := cell.ProcessValues(g.digitComparisons); err != nil {
								t.Error(err)
							}

							assert.Equal(tttt, CellTypeValue, cell.Type())
							assert.Equal(tttt, truth.comparisonValue, cell.comparisonValue)
						})
					}
				}
			})

			tt.Run("placeholders", func(ttt *testing.T) {
				g := GridFromImage(img, fmt.Sprintf("TestGrid_Process_NYT_Comparison_%d", idx+1))
				if err := g.SplitCells(ModeComparison); err != nil {
					t.Error(err)
				}

				for rowIdx, row := range g.Cells {
					for cellIdx, cell := range row {
						truth := truthTableGrid.Cells[rowIdx][cellIdx]
						if truth.Type() != CellTypePlaceholders {
							continue
						}

						ttt.Run(fmt.Sprintf("cell_%s", cell.Identifier), func(tttt *testing.T) {
							if err := cell.ProcessPlaceholders(g.placeholderComparisons); err != nil {
								t.Error(err)
							}
							assert.Equal(tttt, CellTypePlaceholders, cell.Type())
							assert.Equal(tttt, truth.comparisonPlaceholders, cell.comparisonPlaceholders)
						})
					}
				}
			})

			tt.Run("entire grid workers", func(ttt *testing.T) {
				worker := NewGridWorker()
				worker.Start()
				g := GridFromImage(img, fmt.Sprintf("TestGrid_Process_NYT_Comparison_%d", idx+1))
				if err := g.SplitCells(ModeComparison); err != nil {
					t.Error(err)
				}

				if err := g.Process(worker.jobs); err != nil {
					t.Error(err)
				}

				for rowIdx, row := range g.Cells {
					for cellIdx, cell := range row {
						truth := truthTableGrid.Cells[rowIdx][cellIdx]

						ttt.Run(fmt.Sprintf("cell_%s", cell.Identifier), func(tttt *testing.T) {
							switch expression := truth.Type(); expression {
							case CellTypeEmpty:
								assert.Equal(tttt, CellTypeEmpty, cell.Type())
							case CellTypeValue:
								assert.Equal(tttt, CellTypeValue, cell.Type())
								assert.Equal(tttt, truth.comparisonValue, cell.comparisonValue)
							case CellTypePlaceholders:
								assert.Equal(tttt, CellTypePlaceholders, cell.Type())
								assert.Equal(tttt, truth.comparisonPlaceholders, cell.comparisonPlaceholders)
							}
						})
					}
				}
			})
		})
	}
}
