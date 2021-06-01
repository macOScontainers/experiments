package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/macoscontainers/experiments/internal/filesystem"
	"github.com/macoscontainers/experiments/internal/unpack"
	"github.com/macoscontainers/experiments/tests/testutil"
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
	unpacker, err := unpack.UnpackerForImage(sample.RootDir, layersDir)
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
}
