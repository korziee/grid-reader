package internal

import (
	"fmt"
	"image"

	"gopkg.in/gographics/imagick.v3/imagick"
)

type CellType string

const (
	CellTypeValue        CellType = "value"
	CellTypePlaceholders CellType = "placeholders"
	CellTypeEmpty        CellType = "empty"
)

type Mode string

const (
	ModeOCR        Mode = "ocr"
	ModeComparison Mode = "comparison"
)

type Cell struct {
	Identifier string // i.e, R1C1

	image *GridImage
	mode  Mode

	ocrValue        int
	ocrPlaceholders []int

	comparisonValue        int
	comparisonPlaceholders []int
}

func (c *Cell) Type() CellType {
	if c.mode == ModeOCR {
		if c.ocrValue != -1 {
			return CellTypeValue
		}

		if len(c.ocrPlaceholders) > 0 {
			return CellTypePlaceholders
		}

		return CellTypeEmpty
	}

	if c.mode == ModeComparison {
		if c.comparisonValue != -1 {
			return CellTypeValue
		}

		if len(c.comparisonPlaceholders) > 0 {
			return CellTypePlaceholders
		}

		return CellTypeEmpty
	}

	panic("should not reach here")
}

func (c *Cell) Contents() (t CellType, val int, placeholders []int) {
	if c.mode == ModeOCR {
		return c.Type(), c.ocrValue, c.ocrPlaceholders
	}

	return c.Type(), c.comparisonValue, c.comparisonPlaceholders
}

func (c *Cell) ProcessValues(representations []*GridImage) error {
	if err := c.image.RunPreProcessing(); err != nil {
		return fmt.Errorf("running pre-processing on cell: %v", err)
	}

	for repIdx, r := range representations {
		_, distortion := c.image.wand.CompareImages(r.wand, imagick.METRIC_ABSOLUTE_ERROR)
		resolution := r.wand.GetImageWidth() * r.wand.GetImageHeight()
		distortionPercentage := distortion / float64(resolution) * 100

		if distortionPercentage < 5 {
			// TODO: implement a logger
			// fmt.Printf("cell: %s, rep value: %d, distortion%%: %f\n", c.Identifier, repIdx+1, distortionPercentage)
			c.comparisonValue = repIdx + 1
			break
		}
	}

	return nil
}

func (c *Cell) ProcessPlaceholders(representations []*GridImage) error {
	cellBounds := c.image.Image.Bounds()

	offset := 6
	xPos := cellBounds.Min.X + offset
	yPos := cellBounds.Min.Y + offset

	cellPos := 1
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			placeholderRect := image.Rect(
				xPos+(col*39),
				yPos+(row*39),
				xPos+(col*39)+20,
				yPos+(row*39)+25,
			)

			placeholderPosition := row*3 + col + 1

			cropped := NewGridImage(
				c.image.CropImage(placeholderRect),
				fmt.Sprintf("%s/p%d/", c.Identifier, placeholderPosition),
			)

			if err := cropped.RunPreProcessing(); err != nil {
				return fmt.Errorf("running pre-processing on placeholder: %v", err)
			}

			if cropped.wand.GetImageHeight() == 1 || cropped.wand.GetImageWidth() == 1 {
				// image is likely empty after the trim, means the placeholder was empty
				continue
			}

			for repIdx, r := range representations {
				_, distortion := cropped.wand.CompareImages(r.wand, imagick.METRIC_ABSOLUTE_ERROR)
				resolution := r.wand.GetImageWidth() * r.wand.GetImageHeight()
				distortionPercentage := distortion / float64(resolution) * 100

				// if the distortion is less than 20% then we consider it a match
				// note: I was getting success at 5% but it failed on a "6" placeholder
				// on a selected cell
				if distortionPercentage < 20 {
					c.comparisonPlaceholders = append(c.comparisonPlaceholders, repIdx+1)
				}
			}

			cellPos += 1
		}
	}

	return nil
}

func (c *Cell) IdentifyOCR() error {
	if err := c.image.RunPreProcessing(); err != nil {
		return fmt.Errorf("running pre-processing on cell: %v", err)
	}

	c.image.DebugWrite(fmt.Sprintf("ocr/%s.png", c.Identifier))

	identifiedInt, err := c.image.IdentifyIntOCR()
	if err != nil {
		return fmt.Errorf("running identify int: %v", err)
	}
	c.ocrValue = identifiedInt

	placeholders, err := c.image.IdentifyBlockOCR()
	if err != nil {
		return fmt.Errorf("running identify placeholders: %v", err)
	}
	c.ocrPlaceholders = placeholders

	// todo: there is a problem here that if we run OCR (psm = 10)
	// on a single placeholder
	// to solve this will need to get the tsv output which tells us a few things
	// 1. how many characters were identified (and the text)
	// 2. where in the image those characters are (note: need to be careful of the upscaling)
	// 3. the confidence of those captures
	// https: //pkg.go.dev/github.com/Complead/tsv#section-readme

	return nil
}

func NewCellFromGridImage(cellBounds image.Rectangle, img *GridImage, identifier string, mode Mode) *Cell {
	return &Cell{
		Identifier: identifier,
		image:      NewGridImage(img.CropImage(cellBounds), identifier),
		mode:       mode,

		ocrValue:        -1,
		comparisonValue: -1,
	}
}

func NewCell(gridBounds image.Rectangle, img image.Image, identifier string, mode Mode) *Cell {
	return &Cell{
		image:      NewGridImage(img, identifier),
		Identifier: identifier,
		mode:       mode,

		ocrValue:        -1,
		comparisonValue: -1,
	}
}
