package image

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/macoscontainers/experiments/internal/filesystem"
	"github.com/macoscontainers/experiments/internal/layer"
	"github.com/macoscontainers/experiments/internal/marshal"
	archiver "github.com/mholt/archiver/v3"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

// Provides functionality for unpacking OCI container images
type ImageUnpacker struct {
	
	// The directory containing the OCI directory layout for the container image that we will unpack
	imageDir string
	
	// The output directory to which we should unpack filesystem layers
	unpackDir string
	
	// The OCI index for the container image that we will unpack
	index *oci.Index
}

// Creates an ImageUnpacker with the specified options
func UnpackerForImage(imageDir string, unpackDir string) (*ImageUnpacker, error) {
	
	// Attempt to parse the OCI index for the specified container image
	index := &oci.Index{}
	if err := marshal.UnmarshalJsonFile(filepath.Join(imageDir, "index.json"), index); err != nil {
		return nil, err
	}
	
	// Verify that the index contains at least one image manifest
	if len(index.Manifests) < 1 {
		return nil, errors.New("OCI index file did not contain any image manifests")
	}
	
	// Return a populated ImageUnpacker object
	return &ImageUnpacker{
		imageDir: imageDir,
		unpackDir: unpackDir,
		index: index,
	}, nil
}

// Returns an unarchiver object for the archive type denoted by the specified MIME type
func (unpacker *ImageUnpacker) unarchiverForMime(mimetype string) (archiver.Unarchiver, error) {
	switch mimetype {
	
	case "application/vnd.oci.image.layer.v1.tar":
		return archiver.NewTar(), nil
	
	case "application/vnd.oci.image.layer.v1.tar+gzip":
		return archiver.NewTarGz(), nil
	
	default:
		return nil, fmt.Errorf("unsupported archive format %s", mimetype)
	}
}

// Unpacks the filesystem layers in the version of the image for the specified platform, applying the diffs for each successive layer
func (unpacker *ImageUnpacker) Unpack(platform *oci.Platform) (*oci.Manifest, error) {
	
	// Determine whether we are searching for a manifest that matches the a platform or just using the first available manifest
	var descriptor *oci.Descriptor
	if platform == nil {
		
		// Use the first manifest in the index
		descriptor = &unpacker.index.Manifests[0]
		
	} else {
		
		// Search for a manifest that matches the specified platform
		for _, candidate := range unpacker.index.Manifests {
			if candidate.Platform != nil && candidate.Platform.OS == platform.OS && candidate.Platform.Architecture == platform.Architecture {
				descriptor = &candidate
				break
			}
		}
		
		// If no matching manifest was found then stop processing immediately
		if descriptor == nil {
			return nil, fmt.Errorf("could not find image manifest matching platform %s/%s", platform.OS, platform.Architecture)
		}
	}
	
	// Resolve the path to the directory that holds the blobs for the OCI image
	blobsDir := filepath.Join(unpacker.imageDir, "blobs", "sha256")
	
	// Parse the manifest
	manifest := &oci.Manifest{}
	if err := marshal.UnmarshalJsonFile(filepath.Join(blobsDir, descriptor.Digest.Hex()), manifest); err != nil {
		return nil, err
	}
	
	// Unpack each of the filesystem layers in turn
	var previousLayer oci.Descriptor
	for index, layerDetails := range manifest.Layers {
		
		// Resolve the path to the diff and merged directories for the current filesystem layer
		diffDir := filepath.Join(unpacker.unpackDir, layerDetails.Digest.Hex(), "diff")
		mergedDir := filepath.Join(unpacker.unpackDir, layerDetails.Digest.Hex(), "merged")
		
		// Remove the diff directory if it already exists
		if filesystem.Exists(diffDir) {
			if err := os.RemoveAll(diffDir); err != nil {
				return nil, err
			}
		}
		
		// Remove the merged directory if it already exists
		if filesystem.Exists(mergedDir) {
			if err := os.RemoveAll(mergedDir); err != nil {
				return nil, err
			}
		}
		
		/*
		// Retrieve an archive extraction object for the filesystem layer's archive blob
		unarchiver, err := unpacker.unarchiverForMime(layer.MediaType)
		if err != nil {
			return nil, err
		}
		
		// Extract the archive blob for the filesystem layer to the diff directory
		if err := unarchiver.Unarchive(filepath.Join(blobsDir, layer.Digest.Hex()), diffDir); err != nil {
			return nil, err
		}
		*/
		
		// TEMPORARY: use the GNU tar command to perform extraction and preserve attributes
		if err := os.MkdirAll(diffDir, os.ModePerm); err != nil {
			return nil, err
		}
		cmd := exec.Command("tar", "--preserve-permissions", "--same-owner", "-xzvf", filepath.Join(blobsDir, layerDetails.Digest.Hex()), "--directory", diffDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		// Log the command details and run the child process
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		
		// Determine if this is the base filesystem layer
		if index == 0 {
			
			// For the base layer we just symlink the merged directory to the diff directory
			if err := os.Symlink("./diff", mergedDir); err != nil {
				return nil, err
			}
			
		} else {
			
			// Create the merged directory
			if err := os.Mkdir(mergedDir, os.ModePerm); err != nil {
				return nil, err
			}
			
			// Create a DiffApplier for the current layer
			merger := &layer.DiffApplier{
				BaseDir: filepath.Join(unpacker.unpackDir, previousLayer.Digest.Hex(), "merged"),
				DiffDir: diffDir,
				MergedDir: mergedDir,
			}
			
			// Apply the layer's diff to the merged contents of the previous layer
			log.Println("Apply diff", layerDetails.Digest.Hex(), "against base layer", previousLayer.Digest.Hex(), "...")
			errorChannel := merger.ApplyRecursive("", nil, false)
			if err := <-errorChannel; err != nil {
				return nil, err
			}
		}
		
		// Keep track of the previous layer for each loop iteration
		previousLayer = layerDetails
	}
	
	return manifest, nil
}
