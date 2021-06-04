package tests

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/macoscontainers/experiments/internal/filesystem"
	"github.com/macoscontainers/experiments/internal/image"
	"github.com/macoscontainers/experiments/internal/layer"
	"github.com/macoscontainers/experiments/tests/testutil"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

// Removes a file if it exists
func removeIfExists(filename string) error {
	if filesystem.Exists(filename) {
		return os.Remove(filename)
	} else {
		return nil
	}
}

// Tests our code for unpacking OCI container images and applying filesystem layer diffs
func TestUnpack(t *testing.T) {
	
	// Create a SampleLayers object for our test
	sample, err := testutil.SampleImageForTest()
	if err != nil {
		t.Error(err)
		return
	}
	
	// Generate the sample container image that we will unpack
	if err := sample.Generate(); err != nil {
		t.Error(err)
		return
	}
	
	// Create an ImageUnpacker for the sample image
	layersDir := filepath.Join(sample.RootDir, "layers")
	unpacker, err := image.UnpackerForImage(sample.RootDir, layersDir)
	if err != nil {
		t.Error(err)
		return
	}
	
	// Attempt to unpack the image
	manifest, err := unpacker.Unpack(nil)
	if err != nil {
		t.Error(err)
		return
	}
	
	// See if we can round-trip the filesystem layer diffs
	var previousLayer oci.Descriptor
	for index, layerDetails := range manifest.Layers {
		
		// Skip round-tripping for the base layer
		if index == 0 {
			previousLayer = layerDetails
			continue
		}
		
		// Resolve the path to the merged directories for the previous and current filesystem layer
		previousMerged := filepath.Join(layersDir, previousLayer.Digest.Hex(), "merged")
		currentMerged := filepath.Join(layersDir, layerDetails.Digest.Hex(), "merged")
		
		// Create a directory to hold the round-tripped diff
		diffDir := filepath.Join(layersDir, layerDetails.Digest.Hex(), "diff_generated")
		if err := os.MkdirAll(diffDir, os.ModePerm); err != nil {
			t.Error(err)
			return
		}
		
		// Create a DiffGenerator for the current layer
		diff := &layer.DiffGenerator{
			BaseDir: previousMerged,
			ModifiedDir: currentMerged,
			DiffDir: diffDir,
		}
		
		// Generate the layer's diff by comparing it to the merged contents of the previous layer
		log.Println("Diffing", layerDetails.Digest.Hex(), "against base layer", previousLayer.Digest.Hex(), "...")
		errorChannel := diff.DiffRecursive("", nil, false)
		if err := <-errorChannel; err != nil {
			t.Error(err)
			return
		}
		
		// Keep track of the previous layer for each loop iteration
		previousLayer = layerDetails
	}
	
	/*
	// Resolve the paths to the final merged output for the unpacked image and our ground truth filesystem data
	groundTruth := filepath.Join(sample.RootDir, "ground-truth")
	finalLayer := filepath.Join(sample.LayersDir, manifest.Layers[len(manifest.Layers) - 1].Digest.Hex(), "merged")
	
	// Remove file entries that are modified when running an image and therefore should be excluded from our comparison
	excluded := []string{"/.dockerenv", "/dev/console", "/etc/hostname", "/etc/hosts", "/etc/resolv.conf"}
	for _, file := range excluded {
		combinations := []string{filepath.Join(groundTruth, file), filepath.Join(finalLayer, file)}
		for _, combination := range combinations {
			if err := removeIfExists(combination); err != nil {
				t.Error(err)
				return
			}
		}
	}
	
	// Use `git diff` to compare the final merged output to the ground truth data
	groundTruthMount := fmt.Sprintf("-v%s:/hostdir/expected", groundTruth)
	finalLayerMount := fmt.Sprintf("-v%s:/hostdir/actual", finalLayer)
	if err := testutil.DockerRun("git:latest", []string{"diff", "--no-index", "--exit-code", "--name-status", "--no-textconv", "/hostdir/expected", "/hostdir/actual"}, []string{groundTruthMount, finalLayerMount}); err != nil {
		t.Error(err)
	}
	*/
}
