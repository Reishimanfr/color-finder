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

var (
	allowedScaleValues = []string{"1/1", "1/2", "1/4", "1/8", "1/12", "1/16", "1/32", "1/64"}

	pathPtr         = flag.String("path", "", "Path to the image to be analyzed.")
	scalingPtr      = flag.String("scaling", "1/8", "Image scaling used to speed up the program a bit. Available options: 1/1, 1/2, 1/4, 1/8, 1/12, 1/16, 1/32, 1/64")
	threadsPtr      = flag.Int("threads", 20, "Amount of threads to be created by the process.")
	returnAmountPtr = flag.Int("return-amount", 5, "Amount of colors to be returned.")
	debugPtr        = flag.Bool("debug", false, "Should additional data useful for debugging be shown?")
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
	part := make([]byte, 1024)

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

func scaleCoordinates(img image.Image) (uint, uint, error) {
	scaling, err := strconv.Atoi(strings.Split(*scalingPtr, "/")[1])

	if err != nil {
		return 0, 0, err
	}

	scaledX := img.Bounds().Max.X
	scaledY := img.Bounds().Max.Y

	if scaling > 1 {
		scaledX = img.Bounds().Max.X / (scaling / 2)
		scaledY = img.Bounds().Max.Y / (scaling / 2)
	}

	return uint(scaledX), uint(scaledY), nil
}

func main() {
	flag.Parse()

	if !slices.Contains(allowedScaleValues, *scalingPtr) {
		panic("Invalid scaling provided. Expected one of: 1/1, 1/2, 1/4, 1/8, 1/12, 1/16, 1/32, 1/64")
	}

	isDebug := *debugPtr
	start := time.Now()

	imgBuffer := loadImageAsBuffer(pathPtr)
	img, _, err := image.Decode(bytes.NewReader(imgBuffer.Bytes()))

	if err != nil {
		panic(err)
	}

	scaledX, scaledY, err := scaleCoordinates(img)

	if err != nil {
		panic(err)
	}

	tempImage := resize.Resize(scaledX, scaledY, img, resize.Bilinear)
	img = tempImage

	pixelCount := img.Bounds().Max.X * img.Bounds().Max.Y
	colorsMap := make(map[string]int)

	var wg sync.WaitGroup
	var mutex sync.Mutex
	chunkSize := pixelCount / *threadsPtr

	if isDebug {
		fmt.Println("Iterating over", pixelCount, "pixels")
		fmt.Println("Chunk size:", chunkSize)
	}

	for w := 0; w < *threadsPtr; w++ {
		wg.Add(1)

		go func(workerID, startPixel, endPixel int) {
			defer wg.Done()
			localColorsMap := make(map[string]int)
			x := startPixel % img.Bounds().Max.X
			y := startPixel / img.Bounds().Max.X

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
				toString := fmt.Sprintf("%d;%d;%d", rgba.R, rgba.G, rgba.B)

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

	for i := 0; i < *returnAmountPtr; i++ {
		if i >= len(keyValuePairs) {
			break
		}

		rgb := strings.Split(keyValuePairs[i].Key, ";")
		r, g, b := rgb[0], rgb[1], rgb[2]

		formatString := fmt.Sprintf("[\033[38;2;%s;%s;%sm███\033[0;00m] %s,%s,%s -> Seen %d times", r, g, b, r, g, b, keyValuePairs[i].Value)

		fmt.Println(formatString)
	}

	if isDebug {
		duration := time.Since(start)
		fmt.Println("Code took", duration)
	}
}
