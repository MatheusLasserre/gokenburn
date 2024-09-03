package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"sync"
	"time"

	draw "golang.org/x/image/draw"
)

// To use with PNG, change all "jpg" and "jpeg" to "png" and remove the nil from line 69.
// Obs: it uses the bash command to run ffmpeg, so you need to change the command in the absence of bash.
const TargetSizeX = 1920
const TargetSizeY = 1080
const TotalFrames = 120
const Factor = 0.005
const InputPath = "./assets/example.jpg"
const deletedGeneratedFiles = true

func main() {
	input, err := os.Open(InputPath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer input.Close()
	src, err := jpeg.Decode(input)
	if err != nil {
		fmt.Printf("Error decoding file: %v\n", err)
		os.Exit(1)
	}
	// Define biggest image sizes based on the factor
	imageX, imageY := MultiplyFactor(TargetSizeX, Factor, TotalFrames), MultiplyFactor(TargetSizeY, Factor, TotalFrames)
	// Create reference image
	refImage := image.NewRGBA(image.Rect(0, 0, imageX, imageY))
	// Draw the reference image at the final scale using CatmullRom. Check other methods using the intelisense.
	draw.CatmullRom.Scale(refImage, refImage.Rect, src, src.Bounds(), draw.Over, nil)
	// Array to collect the scaled and croped images
	imagesArray := make([]image.Image, TotalFrames)
	start := time.Now()
	// Generate and insert the scaled images in the array
	for i := 0; i < TotalFrames; i++ {
		curFactor := 1 * (PowerFloat(1-Factor, i))
		newImage := image.NewRGBA(image.Rect(0, 0, TargetSizeX, TargetSizeY))
		// Draw the scaled image at the final scale using NearestNeighbor, which is the fastest method.
		draw.NearestNeighbor.Scale(newImage, newImage.Rect, refImage, GetCropBounds(curFactor, imageX, imageY), draw.Over, nil)
		imagesArray[i] = newImage
	}
	fmt.Printf("Each Frame averaged %d milliseconds\n", time.Since(start).Milliseconds()/int64(TotalFrames))

	// Let golang write the files in disk as fast as possible
	wg := &sync.WaitGroup{}
	wg.Add(TotalFrames)
	start = time.Now()
	path, err := os.MkdirTemp("./assets", "example-scaled")
	fmt.Printf("Temp directory: %s\n", path)
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		os.Exit(1)
	}
	for i, v := range imagesArray {
		go func(i int, v image.Image) {
			output, err := os.Create(fmt.Sprintf("%s/example-scaled-%04d.jpg", path, i+1))
			if err != nil {
				fmt.Printf("Error creating file: %v\n", err)
				os.Exit(1)
			}

			jpeg.Encode(output, v, nil)
			output.Close()
			wg.Done()
		}(i, v)
	}
	wg.Wait()
	fmt.Printf("Creating file averaged %d milliseconds\n", time.Since(start).Milliseconds())

	// Send everything to ffmpeg to create the video
	start = time.Now()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("ffmpeg -framerate 30 -start_number 1 -i '%s/example-scaled-%%04d.jpg' -c:v libx264 -pix_fmt yuv420p './assets/example-scaled.mp4'", path))
	fmt.Printf("Command: %s\n", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error creating video: %v\n", err)
		fmt.Printf("Stdout: %s\n", out[:])
		os.Exit(2)
	}
	fmt.Printf("Creating video averaged %d milliseconds\n", time.Since(start).Milliseconds())
	// Delete the temp directory
	if deletedGeneratedFiles {
		os.RemoveAll(path)
	}
}

func MultiplyFactor(d int, factor float64, totalFrames int) int {
	return int(float64(d) * factor * float64(totalFrames))
}

func GetCropBounds(factor float64, outX int, outY int) image.Rectangle {
	finX := int(float64(outX) * factor)
	finY := int(float64(outY) * factor)
	startX := (outX - finX) / 2
	startY := (outY - finY) / 2
	return image.Rect(startX, startY, startX+finX, startY+finY)
}

func PowerFloat(n float64, x int) (result float64) {
	if x == 0 {
		return 1
	}
	if x == 1 {
		return n
	}
	result = n
	for i := 2; i <= x; i++ {
		result = result * n
	}
	return result
}
