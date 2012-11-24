package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"sort"
)

var nocol = flag.Bool("nocol", false, "don't sort columns")
var norow = flag.Bool("norow", false, "don't sort rows")
var passes = flag.Int("p", 1, "the number of passes to make")
var threshold = 0.0

func luma(pixel color.Color) float64 {
	r, g, b, _ := pixel.RGBA()

	luma := .3*float64(r) +
		.59*float64(g) +
		.11*float64(b)

	return luma
}

type SortableColors []color.Color

func (m SortableColors) Len() int {
	return len(m)
}

func (m SortableColors) Swap(i, j int) {
	temp := m[j]
	m[j] = m[i]
	m[i] = temp
}

// this func defines the sort order of the pixels
func (m SortableColors) Less(i, j int) bool {
	return luma(m[i]) > luma(m[j])
}

func rgbaFromImage(src image.Image) (out *image.RGBA) {
	b := src.Bounds()
	out = image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	return
}

func findLumaColBounds(rgba WritableImage, x int, ymin int) (yfirst, ynext int) {
	b := rgba.Bounds()

	yfirst, ynext = -1, b.Max.Y
	for y := ymin; yfirst == -1 && y < b.Max.Y; y++ {
		if luma(rgba.At(x, y)) > threshold {
			yfirst = y
		}
	}

	if yfirst == -1 {
		return
	}

	// then next one above
	for y := yfirst; ynext == b.Max.Y && y < b.Max.Y; y++ {
		if luma(rgba.At(x, y)) < threshold {
			ynext = y
		}
	}
	return
}

func sortCol(rgba WritableImage, x int) {
	b := rgba.Bounds()
	for ymin := 0; ymin < b.Max.Y; {

		yfirst, ynext := findLumaColBounds(rgba, x, ymin)

		if yfirst == -1 {
			break
		}

		// build the to-be-sorted color array
		slice := make([]color.Color, ynext-yfirst)
		for y := yfirst; y < ynext; y++ {
			slice[y-yfirst] = rgba.At(x, y)
		}

		// sort it by luma
		sort.Sort(SortableColors(slice))

		// write it back out
		for y := yfirst; y < ynext; y++ {
			rgba.Set(x, y, slice[y-yfirst])
		}

		ymin = ynext + 1
	}
}

func sortCols(rgba WritableImage) {
	b := rgba.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		sortCol(rgba, x)
	}
	return
}

type WritableImage interface {
	Set(x, y int, c color.Color)
	At(x, y int) color.Color
	Bounds() image.Rectangle
}

// using this to avoid writing sortRows and sortCols
type SwitchyRGBA struct {
	*image.RGBA
	transposed bool
}

func (sr SwitchyRGBA) At(x, y int) (out color.Color) {
	if sr.transposed {
		out = sr.RGBA.At(y, x)
	} else {
		out = sr.RGBA.At(x, y)
	}
	return
}

func (sr SwitchyRGBA) Set(x, y int, c color.Color) {
	if sr.transposed {
		sr.RGBA.Set(y, x, c)
	} else {
		sr.RGBA.Set(x, y, c)
	}
	return
}

func (sr SwitchyRGBA) Bounds() (out image.Rectangle) {
	if sr.transposed {
		b := sr.RGBA.Bounds()
		out = image.Rect(b.Min.Y, b.Min.X, b.Max.Y, b.Max.X)
	} else {
		out = sr.RGBA.Bounds()
	}
	return
}

func pixelSort(img image.Image) image.Image {
	rgba := SwitchyRGBA{rgbaFromImage(img), false}

	if !*norow {
		rgba.transposed = true
		sortCols(rgba)
		rgba.transposed = false
	}
	if !*nocol {
		sortCols(rgba)
	}

	return rgba
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [filename.jpeg] \n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()


	if len(flag.Args())	== 0 {
		flag.Usage()
		os.Exit(-1)
	}
	
	for _, input := range flag.Args() {
		f, err := os.Open(input)
		if err != nil {
			fmt.Printf("Failed to open %s, %v\n", input, err)
			os.Exit(-1)
		}

		img, err := jpeg.Decode(f)
		if err != nil {
			fmt.Printf("Failed to decode %s, %v\n", input, err)
			os.Exit(-1)
		}
		
		
		for threshold = 0; threshold < 55000; threshold += 5000 {
			sortedImg := img
			for i := 0; i < *passes; i++ {
				sortedImg = pixelSort(sortedImg)
			}

			output := fmt.Sprintf("sorted_%dx_%fl_%s", *passes, threshold, input)
			newFile, err := os.Create(output)
			if err != nil {
				fmt.Printf("Failed to write %s", output)
				os.Exit(-1)
			}
			jpeg.Encode(newFile, sortedImg, nil)
		}
	}
}
