package tests

import (
	"path/filepath"
	"testing"

	"github.com/macoscontainers/experiments/internal/unpack"
	"github.com/macoscontainers/experiments/tests/testutil"
)

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
	if err := unpacker.Unpack(nil); err != nil {
		t.Error(err)
	}
}
