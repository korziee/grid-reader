package internal

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

// points are the first non-boundary coord, i.e. topLeft will be the value of the first
// white pixel directly underneath of the top boundary, and directly to the right of the left boundary
type Grid struct {
	img                    *GridImage
	boundaries             image.Rectangle
	separatorThickness     int
	cellLength             int
	placeholderComparisons []*GridImage
	digitComparisons       []*GridImage

	Cells [9][9]*Cell
}

func pixelMeetsThreshold(c color.Color) bool {
	threshold := uint32(100)
	r, g, b, _ := c.RGBA()

	return r < threshold && g < threshold && b < threshold
}

func (g *Grid) SplitCells(cellMode Mode) error {
	midX := g.img.Bounds().Dx() / 2

	// identifies the grid boundaries, cell length and separator thickness
	for y := 0; y < g.img.Bounds().Dy(); y += 1 {
		if !pixelMeetsThreshold(g.img.At(midX, y)) {
			continue
		}

		// process: confirming it's a straight horizontal line and is likely the grid
		//		from the centre, go left and right, until the line ends
		//		if range(leftx, rightx) < 500 (random number) then is not a good top line (go to next candidate)
		//		find the topLeft and topRight coordinates
		//		calculate the width of the top bar
		//		navigate vertically from these coordinates until a value that doesn't pass the threshold is met
		//		this is the bottom of the grid
		//		set bottomLeft and bottomRight coordinates
		//		validate that all points share a euclidean distance

		var topLeft, topRight, bottomRight image.Point
		var separatorThickness int

		leftX := midX - 1
		rightX := midX + 1
		for x := leftX; x >= 0; x -= 1 {
			if !pixelMeetsThreshold(g.img.At(x, y)) {
				topLeft = image.Point{x + 1, y}
				break
			}
		}

		for x := rightX; x <= g.img.Bounds().Dx(); x += 1 {
			if !pixelMeetsThreshold(g.img.At(x, y)) {
				topRight = image.Point{x - 1, y}
				break
			}
		}

		if topRight.Sub(topLeft).X < 500 {
			// need to keep looking downwards
			fmt.Println("Not a good top line candidate, was under 500px")
			continue
		}

		for yDown := y + 1; yDown < g.img.Bounds().Dy(); yDown += 1 {
			leftPixelMetThreshold := pixelMeetsThreshold(g.img.At(topLeft.X, yDown))
			rightPixelMetThreshold := pixelMeetsThreshold(g.img.At(topRight.X, yDown))

			if !leftPixelMetThreshold && !rightPixelMetThreshold {
				bottomRight = image.Point{topRight.X, yDown}
				break
			}

			if leftPixelMetThreshold != rightPixelMetThreshold {
				return fmt.Errorf("grid top points appear unaligned, expected both verticals to end at the same y coord")
			}
		}

		for thick := 1; thick < 100; thick += 1 {
			if !pixelMeetsThreshold(g.img.At(midX, y+thick)) {
				separatorThickness = thick
				break
			}
		}

		if separatorThickness > 50 {
			return fmt.Errorf("found a large separator thickness, expected less than 50px")
		}

		g.cellLength =
			(topRight.X -
				topLeft.X -
				// *4 cause theres the two sides + the two middle dividers.
				(separatorThickness * 4) -
				// *6 here to cater for the the thin divers between cells
				// 9 cause there are nine columns. likely float is being cast to int, its ok
				((separatorThickness / 2) * 6)) / 9

		g.separatorThickness = separatorThickness
		g.boundaries = image.Rect(topLeft.X, topLeft.Y, bottomRight.X, bottomRight.Y)
		break
	}

	if i := g.boundaries.Size(); i.X == 0 || i.Y == 0 {
		return fmt.Errorf("failed to find grid boundaries")
	}

	g.img.DebugWrite("grid.png")

	yPos := g.boundaries.Min.Y + g.separatorThickness
	for row := 0; row < 9; row += 1 {
		var rowCells [9]*Cell

		xPos := g.boundaries.Min.X + g.separatorThickness
		for col := 0; col < 9; col += 1 {
			bounds := image.Rect(
				// + 2 to give some buffer from the borders as pixels might be changing colour
				xPos+2,
				yPos+2,

				// - 2 to compensate for operation above
				xPos+g.cellLength-2,
				yPos+g.cellLength-2,
			)

			rowCells[col] = NewCellFromGridImage(
				bounds,
				g.img,
				fmt.Sprintf("R%dC%d", row+1, col+1),
				cellMode,
			)

			xPos += g.cellLength
			// set offset
			switch col {
			// on the 3th and 6th increase by the entire separator thickness
			case 2:
				xPos += g.separatorThickness
			case 5:
				xPos += g.separatorThickness
			default:
				// increase xPos by cellLength and grid.separatorThickness/2 each time
				xPos += g.separatorThickness / 2
			}
		}

		g.Cells[row] = rowCells

		yPos += g.cellLength
		// set row offset
		switch row {
		// on the 3th and 6th increase by the entire separator thickness
		case 2:
			yPos += g.separatorThickness
		case 5:
			yPos += g.separatorThickness
		default:
			// increase yPos by cellLength and grid.separatorThickness/2 each time
			yPos += g.separatorThickness / 2
		}
	}

	return nil
}

func (g *Grid) Process(jobs chan<- *WorkerJob) error {
	results := make(chan *Result, 81)

	for _, columns := range g.Cells {
		for _, c := range columns {
			jobs <- &WorkerJob{cell: c, grid: g, res: results}
		}
	}

	for range 81 {
		msg := <-results
		if !msg.Ok {
			return msg.Error
		}
	}

	return nil
}

// TODO: TEST
// Returns a continuous string containing the contents of each
// cell, left to right, top to bottom. Empty cells are represented
// by a period (.). Placeholders are omitted.
func (g *Grid) String() string {
	var str string

	for _, row := range g.Cells {
		for _, cell := range row {
			if cell.Type() == CellTypeValue {
				str += fmt.Sprintf("%d", cell.comparisonValue)
			} else {
				str += "."
			}
		}
	}

	return str
}

func loadPlaceholderComparisons() []*GridImage {
	placeholderComparisons := make([]*GridImage, 0)

	// this gets back the path of the current file, I tried with os.Getwd() but that
	// returns the location that the binary was called from
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("could not determine the current file path")
	}

	baseDir := filepath.Dir(currentFile)
	path := path.Join(baseDir, "../t-placeholders")
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(fmt.Errorf("reading placeholders directory: %w", err))
	}

	for idx, e := range entries {
		file, err := os.Open(fmt.Sprintf("%s/%s", path, e.Name()))
		if err != nil {
			log.Fatal(fmt.Errorf("opening image file: %v", err))
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			log.Fatal(fmt.Errorf("decoding image: %v", err))
		}

		placeholderComparisons = append(placeholderComparisons, NewGridImage(img, fmt.Sprintf("placeholder-%d", idx)))
	}

	return placeholderComparisons
}

func loadDigitComparisons() []*GridImage {
	digitComparisons := make([]*GridImage, 0)

	// this gets back the path of the current file, I tried with os.Getwd() but that
	// returns the location that the binary was called from
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("could not determine the current file path")
	}

	baseDir := filepath.Dir(currentFile)
	path := path.Join(baseDir, "../t-values")
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(fmt.Errorf("reading values directory: %w", err))
	}

	for idx, e := range entries {
		file, err := os.Open(fmt.Sprintf("%s/%s", path, e.Name()))
		if err != nil {
			log.Fatal(fmt.Errorf("opening image file: %v", err))
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			log.Fatal(fmt.Errorf("decoding image: %v", err))
		}

		digitComparisons = append(digitComparisons, NewGridImage(img, fmt.Sprintf("digit-%d", idx)))
	}

	return digitComparisons
}

func GridFromImage(img image.Image) *Grid {
	placeholderComparisons := loadPlaceholderComparisons()
	digitComparisons := loadDigitComparisons()
	return &Grid{
		img:                    NewGridImage(img, "grid"),
		placeholderComparisons: placeholderComparisons,
		digitComparisons:       digitComparisons,
	}
}
