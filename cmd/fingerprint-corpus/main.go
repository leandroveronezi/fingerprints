package main

import (
	"image"
	"image/png"
	"log"
	"os"
	"path"

	"github.com/alevinval/fingerprints/internal/debug"
	"github.com/alevinval/fingerprints/internal/extraction"
	"github.com/alevinval/fingerprints/internal/helpers"
	"github.com/alevinval/fingerprints/internal/kernel"
	"github.com/alevinval/fingerprints/internal/matrix"
	"github.com/alevinval/fingerprints/internal/processing"
)

var outFolderPath = "out"

func showImage(title string, img image.Image) {
	f, err := os.Create(path.Join(outFolderPath, title+".png"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	png.Encode(f, img)
}

func showMatrix(title string, m *matrix.M) {
	showImage(title, m.ToGray())
}

func processImage(img image.Image, in *matrix.M) {
	bounds := in.Bounds()
	normalized := matrix.New(bounds)

	metaIn := processing.Metadata(in)
	processing.Normalize(in, normalized, metaIn)
	showMatrix("Normalized", normalized)

	gx, gy := matrix.New(bounds), matrix.New(bounds)
	kernel.SobelDx.ConvoluteParallelized(normalized, gx)
	kernel.SobelDy.ConvoluteParallelized(normalized, gy)

	// Compute filtered directional
	filteredD, normFilteredD := matrix.New(bounds), matrix.New(bounds)
	kernel.FilteredDirectional(gx, gy, 4).ConvoluteParallelized(filteredD, filteredD)
	metaFilteredD := processing.Metadata(filteredD)
	processing.Normalize(filteredD, normFilteredD, metaFilteredD)
	showMatrix("Filtered Directional", normFilteredD)

	// Compute segmented image
	segmented, normSegmented := matrix.New(bounds), matrix.New(bounds)
	kernel.Variance(filteredD).ConvoluteParallelized(normalized, segmented)
	metaSegmented := processing.Metadata(segmented)
	processing.Normalize(segmented, normSegmented, metaSegmented)
	showMatrix("Filtered Directional Std Dev.", normSegmented)

	// Compute binarized segmented image
	binarizedSegmented, binarizedSegmentedNorm := matrix.New(bounds), matrix.New(bounds)
	processing.Binarize(normSegmented, binarizedSegmented, metaIn)
	processing.BinarizeEnhancement(binarizedSegmented)
	metaBinarizedSegmented := processing.Metadata(binarizedSegmented)
	processing.Normalize(binarizedSegmented, binarizedSegmentedNorm, metaBinarizedSegmented)
	showMatrix("Binarized Segmented", binarizedSegmentedNorm)

	// Binarize normalized image
	skeletonized := matrix.New(bounds)
	processing.Binarize(normalized, skeletonized, metaIn)
	processing.BinarizeEnhancement(skeletonized)
	processing.Skeletonize(skeletonized)
	showMatrix("Skeletonized", skeletonized)

	// Run the whole thing again, the steps above are just intermediates
	// so we can have visibility on the algorithms
	result := extraction.DetectionResult(in)

	debug.DrawFeatures(img, result)
	showImage("Debug", img)
}

func ensurePathExists(path string) {
	err := os.Mkdir(path, os.ModePerm)
	if err != nil {
		log.Printf("error creating output path: %s", err)
	}
}

func main() {
	if len(os.Args) < 3 {
		println("usage: ./fingerprint-corpus [input-image] [output-folder]")
		return
	}

	inputImagePath := os.Args[1]
	outFolderPath = os.Args[2]

	log.SetFlags(log.Flags() + log.Lshortfile)
	ensurePathExists(outFolderPath)
	img, m := helpers.LoadImage(inputImagePath)
	processImage(img, m)
}
