package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"sort"
)

var nocol = flag.Bool("nocol", false, "don't sort columns")
var norow = flag.Bool("norow", false, "don't sort rows")
var passes = flag.Int("p", 1, "the number of passes to make")

var maxThreshold = flag.Float64("tmax", 60000, "max luma threshold")
var minThreshold = flag.Float64("tmin", 0, "min luma threshold")
var thresholdInc = flag.Float64("tinc", 5000, "threshold increment amount")
var threshold = 0.0

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [files...]\n",
			os.Args[0])
		flag.PrintDefaults()
	}

}

func luma(pixel color.Color) float64 {
	r, g, b, _ := pixel.RGBA()

	return .2126*float64(r) +
		.7152*float64(g) +
		.0722*float64(b)
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

func (m SortableColors) Less(i, j int) bool {
	return luma(m[i]) > luma(m[j])
}

func rgbaFromImage(src image.Image) (out *image.RGBA) {
	b := src.Bounds()
	out = image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	return
}

func sortSequence(seq []color.Color) {
	for i := 0; i < len(seq); {
		runEnd := len(seq)
		foundRun := false

		for j := i; j < len(seq); j++ {
			if luma(seq[j]) > threshold {
				foundRun = true
				i = j
				for ; j < len(seq); j++ {
					if luma(seq[j]) < threshold {
						runEnd = j
						break
					}
				}
				break
			}
		}

		if !foundRun {
			break
		}

		run := seq[i:runEnd]
		sort.Sort(SortableColors(run))

		i = runEnd + 1
	}
}

func pixelSort(img image.Image) image.Image {
	rgba := rgbaFromImage(img)
	b := rgba.Bounds()

	if !*norow {
		seq := make([]color.Color, b.Max.X)
		for y := 0; y < b.Max.Y; y++ {
			for i := range seq {
				seq[i] = rgba.At(i, y)
			}
			sortSequence(seq)
			for i := range seq {
				rgba.Set(i, y, seq[i])
			}
		}
	}

	if !*nocol {
		seq := make([]color.Color, b.Max.Y)
		for x := 0; x < b.Max.X; x++ {
			for i := range seq {
				seq[i] = rgba.At(x, i)
			}
			sortSequence(seq)
			for i := range seq {
				rgba.Set(x, i, seq[i])
			}
		}
	}
	return rgba
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(-1)
	}

	for _, input := range flag.Args() {
		f, err := os.Open(input)
		if err != nil {
			fmt.Printf("Failed to open %s, %v\n", input, err)
			os.Exit(-1)
		}

		img, format, err := image.Decode(f)
		if err != nil {
			fmt.Printf("Failed to decode %s, %v, %v\n", input, format, err)
			os.Exit(-1)
		}

		for threshold = *minThreshold; threshold < *maxThreshold; threshold += *thresholdInc {
			sortedImg := img
			for i := 0; i < *passes; i++ {
				sortedImg = pixelSort(sortedImg)
			}

			output := fmt.Sprintf("sorted_%dx_%fl_%s", *passes, threshold, input)
			if format == "gif" {
				output = fmt.Sprint(output[:3], "jpeg")
			}

			newFile, err := os.Create(output)
			if err != nil {
				fmt.Printf("Failed to write %s", output)
				os.Exit(-1)
			}

			switch format {
			case "jpeg":
				jpeg.Encode(newFile, sortedImg, nil)
			case "png":
				png.Encode(newFile, sortedImg)
			case "gif":
				jpeg.Encode(newFile, sortedImg, nil)
			default:
				fmt.Printf("Unsupported image format %v\n", format)
				os.Exit(-1)
			}
		}
	}
}
