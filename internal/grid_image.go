package internal

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"gopkg.in/gographics/imagick.v3/imagick"
)

type GridImage struct {
	image.Image
	identifier string
	wand       *imagick.MagickWand
}

func (g *GridImage) Bytes() ([]byte, error) {
	b, err := g.wand.GetImageBlob()
	if err != nil {
		return nil, fmt.Errorf("failed to get bytes: %v", err)
	}

	return b, nil
}

/*
*

	// magick debug/ocr/R3C4.png \
	// -fill none \
	// -opaque 'rgb(255,255,255)' \
	// -alpha extract \
	// -threshold 0 \
	// -negate \
	// -transparent white \
	// -trim \
	// r3c4.png
*/

// TODO: explain what this does
func (g *GridImage) RunPreProcessing() error {
	g.DebugWrite(fmt.Sprintf("%s/%s", g.identifier, "0-original.png"))
	if err := g.wand.SetImageType(imagick.IMAGE_TYPE_GRAYSCALE); err != nil {
		return fmt.Errorf("setting image type to grayscale: %v", err)
	}

	g.DebugWrite(fmt.Sprintf("%s/%s", g.identifier, "1-gray.png"))

	var medianCount uint
	var medianPixelInfo *imagick.PixelInfo

	resolution, histogram := g.wand.GetImageHistogram()
	for _, h := range histogram {
		if h.GetColorCount() > medianCount {
			medianCount = h.GetColorCount()
			medianPixelInfo = h.GetMagickColor()
		}
	}

	if medianCount < resolution/2 {
		return fmt.Errorf("failed to get median colour, count: %d, resolution: %v", medianCount, resolution)
	}

	background := imagick.NewPixelWand()
	background.SetPixelColor(medianPixelInfo)

	replacer := imagick.NewPixelWand()
	replacer.SetAlpha(0)
	replacer.SetColor("none")

	if err := g.wand.OpaquePaintImage(background, replacer, 0.0, false); err != nil {
		return fmt.Errorf("opaque paint image: %v", err)
	}
	g.DebugWrite(fmt.Sprintf("%s/%s", g.identifier, "2-bg-paint.png"))

	// todo: explain the manual pixel iteration
	pixelIterator := g.wand.NewPixelIterator()
	for y := 0; y < int(g.wand.GetImageHeight()); y++ {
		row := pixelIterator.GetNextIteratorRow()
		if row == nil {
			return fmt.Errorf("failed to get pixel row at y=%d", y)
		}

		for _, pixel := range row {
			if pixel.GetAlpha() == 0 {
				pixel.SetColor("white")
			} else {
				pixel.SetColor("black")
			}
		}

		if err := pixelIterator.SyncIterator(); err != nil {
			return fmt.Errorf("failed to sync iterator at y=%d: %v", y, err)
		}
	}
	g.DebugWrite(fmt.Sprintf("%s/%s", g.identifier, "3-after-pix-ter.png"))

	bg := imagick.NewPixelWand()
	bg.SetColor("white")

	if err := g.wand.TransparentPaintImage(bg, 0, 0, false); err != nil {
		return fmt.Errorf("transparent paint image: %v", err)
	}
	g.DebugWrite(fmt.Sprintf("%s/%s", g.identifier, "4-transparent-paint.png"))

	if err := g.wand.TrimImage(0.0); err != nil {
		return fmt.Errorf("trimming image: %v", err)
	}
	g.DebugWrite(fmt.Sprintf("%s/%s", g.identifier, "5-trim-final.png"))

	return nil
}

func (g *GridImage) tesseract(psm int) (string, error) {
	// "-" is stdin
	cmd := exec.Command("tesseract", "stdin", "stdout", "--psm", strconv.Itoa(psm), "quiet")

	// tesseract might pick up a newlines on single digit (PSM = 10)
	// which aren't in the whitelist, which means nothing is returned
	// semi odd behaviour, but we validate on the way out anyway
	if psm != 10 {
		cmd.Args = append(cmd.Args, "-c", "tessedit_char_whitelist=123456789")
	}

	b, err := g.Bytes()
	if err != nil {
		return "", fmt.Errorf("getting bytes: %v", err)
	}

	cmd.Stdin = bytes.NewReader(b)

	// is there a better way to do this?
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running tesseract: %v", err)
	}

	if len(stderr.Bytes()) != 0 {
		return "", fmt.Errorf("tesseract failed: %s", stderr.String())
	}

	return stdout.String(), nil
}

func (g *GridImage) IdentifyIntOCR() (int, error) {
	//  10|single_char             Treat the image as a single character.
	out, err := g.tesseract(10)
	if err != nil {
		return -1, fmt.Errorf("identifying int: %v", err)
	}

	cleaned := strings.TrimSpace(out)
	if cleaned == "" || cleaned == "_" {
		return -1, nil
	}

	val, err := strconv.Atoi(cleaned)
	if err != nil || val > 9 {
		return -1, nil
	}

	return val, nil
}

func (g *GridImage) IdentifyBlockOCR() (ps []int, e error) {
	// 6|single_block            Assume a single uniform block of text.
	out, err := g.tesseract(6)
	if err != nil {
		return ps, fmt.Errorf("running tesseract: %v", err)
	}

	ints := make([]int, 0)
	for _, c := range out {
		if c == ' ' || c == '\n' {
			continue
		}
		i, err := strconv.Atoi(string(c))
		if err != nil {
			return ps, fmt.Errorf("unable to convert block value to integer: %v", err)
		}
		ints = append(ints, i)
	}

	return ints, nil
}

func (g *GridImage) DebugWrite(p string) {
	if os.Getenv("DEBUG") != "true" {
		return
	}

	if g.wand.GetImageHeight() == 1 || g.wand.GetImageWidth() == 1 {
		// image is likely empty, will error getting bytes
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("getting current directory: %w", err))
	}

	filePath := path.Join(currentDir, "../debug", p)

	if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
		log.Fatal(fmt.Errorf("creating directory: %w", err))
	}

	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create debug file: %w", err))
	}
	defer f.Close()

	b, err := g.Bytes()
	if err != nil {
		log.Fatal(fmt.Errorf("getting bytes: %w", err))
	}

	if _, err = f.Write(b); err != nil {
		log.Fatal(fmt.Errorf("failed to write debug image: %w", err))
	}
}

func NewGridImage(img image.Image, identifier string) *GridImage {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(fmt.Errorf("failed to encode PNG: %w", err))
	}

	wand := imagick.NewMagickWand()
	if err := wand.ReadImageBlob(buf.Bytes()); err != nil {
		log.Fatal(fmt.Errorf("reading image blob: %w", err))
	}

	return &GridImage{
		Image:      img,
		wand:       wand,
		identifier: identifier,
	}
}
