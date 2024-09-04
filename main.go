package main

import (
	"fmt"
	"image"

	// "image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/goki/freetype"
	draw "golang.org/x/image/draw"
	"golang.org/x/image/font"
)

// To use with PNG, change all "jpg" and "jpeg" to "png" and remove the nil from line 69.
// Obs: it uses the bash command to run ffmpeg, so you need to change the command in the absence of bash.
const InputPath = "./assets/example.jpg"
const TotalFrames = 10

var text = []string{"Hello", "World"}

const fontPath = "./assets/Helvetica.ttf"
const fontSize = 96
const fontSpacing = 1

var color = RGBA{
	R: 255,
	G: 255,
	B: 255,
	A: 255,
}

const LastFrameTotalFactor = 1.1 // WIP: e.g: If you want to zoom until the image is 110% of the original, set this to 0.1
const Factor = 0.02
const TargetSizeX = 1920
const TargetSizeY = 1080
const deleteGeneratedFiles = false
const generateVideo = false
const TextPercentDx float64 = 0.1
const TextPercentDy float64 = 0.8

var TextPosX = TargetSizeX * TextPercentDx
var TextPosY = (TargetSizeY*TextPercentDy + fontSize*2/3)

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
	fmt.Printf("Image size: %dx%d\n", imageX, imageY)
	// Create reference image
	refImage := image.NewRGBA(image.Rect(0, 0, imageX, imageY))
	// Draw the reference image at the final scale using CatmullRom. Check other methods using the intelisense.
	draw.CatmullRom.Scale(refImage, refImage.Rect, src, src.Bounds(), draw.Over, nil)

	// Load font
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		fmt.Printf("Error reading font file: %v\n", err)
		os.Exit(1)
	}
	// Parse font
	fnt, err := freetype.ParseFont(fontBytes)
	if err != nil {
		fmt.Printf("Error parsing font: %v\n", err)
		os.Exit(1)
	}
	//
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(fnt)
	c.SetFontSize(fontSize)
	c.SetHinting(font.HintingFull)
	c.SetClip(image.Rect(0, 0, TargetSizeX, TargetSizeY))
	c.SetSrc(image.NewUniform(color))

	// Array to collect the scaled and croped images
	imagesArray := make([]image.Image, TotalFrames)
	start := time.Now()
	// Generate and insert the scaled images in the array
	for i := 0; i < TotalFrames; i++ {
		curFactor := 1 * (PowerFloat(1-Factor, i))
		newImage := image.NewRGBA(image.Rect(0, 0, TargetSizeX, TargetSizeY))
		// Draw the scaled image at the final scale using NearestNeighbor, which is the fastest method.
		draw.NearestNeighbor.Scale(newImage, newImage.Rect, refImage, GetCropBounds(curFactor, imageX, imageY), draw.Over, nil)
		// Draw the text on the image
		c.SetDst(newImage)
		// pt := freetype.Pt(10, 10+int(c.PointToFixed(fontSize)>>6))
		pt := freetype.Pt(int(TextPosX), int(TextPosY))
		for _, s := range text {
			_, err = c.DrawString(s, pt)
			if err != nil {
				fmt.Printf("Error drawing text: %v\n", err)
				os.Exit(1)
			}
			pt.Y += c.PointToFixed(fontSize * fontSpacing)
		}
		imagesArray[i] = newImage
	}
	fmt.Printf("Each Frame averaged %d milliseconds\n", time.Since(start).Milliseconds()/int64(TotalFrames))

	// Let golang write the files in disk as fast as possible
	path := WriteFiles(imagesArray)

	if generateVideo {
		GenerateAndSaveVideo(path, deleteGeneratedFiles)
	}
}

func MultiplyFactor(d int, factor float64, totalFrames int) int {
	return int(float64(d) * PowerFloat(1+factor, totalFrames))
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

func GenerateAndSaveVideo(path string, deleteGeneratedFiles bool) {
	// Send everything to ffmpeg to create the video
	start := time.Now()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("ffmpeg -framerate 30 -start_number 1 -i '%s/example-scaled-%%04d.jpg' -c:v libx264 -pix_fmt yuv420p './assets/example-scaled.mp4'", path))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error creating video: %v\n", err)
		fmt.Printf("Stdout: %s\n", out[:])
		os.Exit(2)
	}
	fmt.Printf("Creating video averaged %d milliseconds\n", time.Since(start).Milliseconds())
	// Delete the temp directory
	if deleteGeneratedFiles {
		os.RemoveAll(path)
	}
}

func WriteFiles(imagesArray []image.Image) string {
	wg := &sync.WaitGroup{}
	wg.Add(TotalFrames)
	path, err := os.MkdirTemp("./assets/tmp", "tmp-images")
	fmt.Printf("Temp directory: %s\n", path)
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		os.Exit(1)
	}
	start := time.Now()
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

	return path
}

type Color interface {
	// RGBA returns the alpha-premultiplied red, green, blue and alpha values
	// for the color. Each value ranges within [0, 0xffff], but is represented
	// by a uint32 so that multiplying by a blend factor up to 0xffff will not
	// overflow.
	//
	// An alpha-premultiplied color component c has been scaled by alpha (a),
	// so has valid values 0 <= c <= a.
	RGBA() (r, g, b, a uint32)
}

// RGBA represents a traditional 32-bit alpha-premultiplied color, having 8
// bits for each of red, green, blue and alpha.
//
// An alpha-premultiplied color component C has been scaled by alpha (A), so
// has valid values 0 <= C <= A.
type RGBA struct {
	R, G, B, A uint8
}

func (c RGBA) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.B)
	b |= b << 8
	a = uint32(c.A)
	a |= a << 8
	return
}
