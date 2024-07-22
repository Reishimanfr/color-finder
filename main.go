package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nfnt/resize"
)

const (
	chunksize = 1024 //todo
	offset    = 10   //todo

)

var (
	allowedScaleValues = []string{"1/1", "1/2", "1/4", "1/8", "1/12", "1/16", "1/32"}

	pathPtr    = flag.String("path", "", "Path to the image to be analyzed.")
	offsetPtr  = flag.Int("offset", 10, "Offset used for RGB colors to decrease similar colors appearing.")
	scalingPtr = flag.String("scaling", "1/4", "Image scaling used to speed up the program a bit. Available options: 1/1, 1/2, 1/4, 1/8, 1/12, 1/16, 1/32")
	threadsPtr = flag.Int("threads", 20, "Amount of threads to be created by the process.")
)

type KeyValue struct {
	Key   string
	Value int
}

func loadImageAsBuffer(path *string) *bytes.Buffer {
	data, err := os.Open(*path)

	if err != nil {
		panic(err)
	}

	defer data.Close()

	reader := bufio.NewReader(data)
	buffer := bytes.NewBuffer(make([]byte, 0))
	part := make([]byte, chunksize)

	var count int

	for {
		if count, err = reader.Read(part); err != nil {
			break
		}
		buffer.Write(part[:count])
	}

	if err != io.EOF {
		panic(err)
	} else {
		err = nil
	}

	return buffer
}

// Returns 2 booleans for both X and Y
func checkIfOutOfBounds(x int, y int, bounds image.Rectangle) (bool, bool) {
	xOut := false
	yOut := false

	if x < bounds.Min.X || x >= bounds.Max.X {
		xOut = true
	}

	if y < bounds.Min.Y || y >= bounds.Max.Y {
		yOut = true
	}

	return xOut, yOut
}

func main() {
	flag.Parse()

	if !slices.Contains(allowedScaleValues, *scalingPtr) {
		panic("Invalid scaling provided. Expected one of: 1/1, 1/2, 1/4, 1/8, 1/12, 1/16, 1/32")
	}

	start := time.Now()

	imgBuffer := loadImageAsBuffer(pathPtr)
	img, _, err := image.Decode(bytes.NewReader(imgBuffer.Bytes()))

	if err != nil {
		panic(err)
	}

	scaling, err := strconv.Atoi(strings.Split(*scalingPtr, "/")[1])

	if err != nil {
		panic(err)
	}

	scaledX := img.Bounds().Max.X
	scaledY := img.Bounds().Max.Y

	if scaling > 1 {
		scaledX = img.Bounds().Max.X / (scaling / 2)
		scaledY = img.Bounds().Max.Y / (scaling / 2)
	}

	tempImage := resize.Resize(uint(scaledX), uint(scaledY), img, resize.Bilinear)
	img = tempImage

	pixelCount := img.Bounds().Max.X * img.Bounds().Max.Y
	colorsMap := make(map[string]int)

	fmt.Println("Iterating over", pixelCount, "pixels")

	var wg sync.WaitGroup
	var mutex sync.Mutex
	chunkSize := pixelCount / *threadsPtr

	fmt.Println("Chunk size:", chunkSize)

	for w := 0; w < *threadsPtr; w++ {
		wg.Add(1)

		go func(workerID, startPixel, endPixel int) {
			defer wg.Done()
			localColorsMap := make(map[string]int)
			x, y := startPixel%img.Bounds().Max.X, startPixel/img.Bounds().Max.X

			for i := startPixel; i < endPixel; i++ {
				xOut, yOut := checkIfOutOfBounds(x, y, img.Bounds())

				if xOut {
					x = 0
					y++
				}

				if yOut {
					panic("Y coordinate out of bounds")
				}

				rgba := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				toString := fmt.Sprintf("%d,%d,%d", rgba.R, rgba.G, rgba.B)

				localColorsMap[toString]++
				x++
			}

			mutex.Lock()
			for k, v := range localColorsMap {
				colorsMap[k] += v
			}
			mutex.Unlock()
		}(w, w*chunkSize, (w+1)*chunkSize)
	}

	wg.Wait()

	var keyValuePairs []KeyValue

	for k, v := range colorsMap {
		keyValuePairs = append(keyValuePairs, KeyValue{k, v})
	}

	sort.Slice(keyValuePairs, func(i, j int) bool {
		return keyValuePairs[i].Value > keyValuePairs[j].Value
	})

	for i := 0; i < 10; i++ {
		fmt.Println("RGB:", keyValuePairs[i].Key, ":", keyValuePairs[i].Value, "times")
	}

	duration := time.Since(start)

	fmt.Println("Code took ", duration)
}
