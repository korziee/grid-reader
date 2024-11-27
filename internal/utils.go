package internal

import (
	"image"
)

func CropImage(img image.Image, rect image.Rectangle) image.Image {
	subImg := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(rect)

	return subImg
}
