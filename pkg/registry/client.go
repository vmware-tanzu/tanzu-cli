// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"archive/tar"
	"bytes"
	"io"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/cmd"
	ctlimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

type registry struct {
	opts     *ctlimg.Opts
	registry ctlimg.Registry
}

// New instantiates a new Registry
func New(opts *ctlimg.Opts) (Registry, error) {
	reg, err := ctlimg.NewSimpleRegistry(*opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize registry client")
	}

	return &registry{
		opts:     opts,
		registry: reg,
	}, nil
}

// ListImageTags lists all tags of the given image.
func (r *registry) ListImageTags(imageName string) ([]string, error) {
	ref, err := regname.ParseReference(imageName, regname.WeakValidation)
	if err != nil {
		return []string{}, err
	}

	return r.registry.ListTags(ref.Context())
}

// GetFile gets the file content bundled in the given image:tag.
// If filename is empty, it will get the first file.
func (r *registry) GetFile(imageWithTag, filename string) ([]byte, error) {
	ref, err := regname.ParseReference(imageWithTag, regname.WeakValidation)
	if err != nil {
		return nil, err
	}
	d, err := r.registry.Get(ref)
	if err != nil {
		return nil, errors.Wrap(err, "Collecting images")
	}

	img, err := d.Image()
	if err != nil {
		return nil, err
	}

	return getFileContentFromImage(img, filename)
}

func getFileContentFromImage(image regv1.Image, filename string) ([]byte, error) {
	layers, err := image.Layers()

	if err != nil {
		return nil, err
	}

	for _, imgLayer := range layers {
		files, err := getFilesFromLayer(imgLayer)
		if err != nil {
			return nil, err
		}
		for k, v := range files {
			if filename == "" || k == filename {
				return v, nil
			}
		}
	}
	return nil, errors.New("cannot find file from the image")
}

func getFilesFromLayer(imgLayer regv1.Layer) (map[string][]byte, error) {
	layerStream, err := imgLayer.Uncompressed()
	if err != nil {
		return nil, err
	}
	defer layerStream.Close()

	files := make(map[string][]byte)
	tarReader := tar.NewReader(layerStream)
	for {
		hdr, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return files, err
		}
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA { //nolint:staticcheck //SA1019: tar.TypeRegA has been deprecated since Go 1.11 and an alternative has been available since Go 1.1: Use TypeReg instead. (staticcheck)
			buf, err := io.ReadAll(tarReader)
			if err != nil {
				return files, err
			}
			files[hdr.Name] = buf
		}
	}
	return files, nil
}

// GetFiles get all the files content bundled in the given image:tag.
func (r *registry) GetFiles(imageWithTag string) (map[string][]byte, error) {
	ref, err := regname.ParseReference(imageWithTag, regname.WeakValidation)
	if err != nil {
		return nil, err
	}
	d, err := r.registry.Get(ref)
	if err != nil {
		return nil, errors.Wrap(err, "Collecting images")
	}
	img, err := d.Image()
	if err != nil {
		return nil, err
	}

	return getAllFilesContentFromImage(img)
}

func getAllFilesContentFromImage(image regv1.Image) (map[string][]byte, error) {
	layers, err := image.Layers()

	if err != nil {
		return nil, err
	}

	var files map[string][]byte
	for _, imgLayer := range layers {
		files, err = getFilesFromLayer(imgLayer)
		if err != nil {
			return nil, err
		}
	}
	if len(files) != 0 {
		return files, nil
	}
	return nil, errors.New("cannot find file from the image")
}

// DownloadBundle downloads OCI bundle similar to `imgpkg pull -b` command
// It is recommended to use this function when downloading imgpkg bundle because
//   - During the air-gapped script, these plugin discovery packages are copied to a
//     private registry with the `imgpkg copy` command
//   - Downloading files directly from OCI image similar to `GetFiles` doesn't work
//     because it doesn't update the `ImageLock` file when we download the package from
//     different registry. And returns original ImageLock file. and as ImageLock file
//     is pointing to original registry instead of private registry, image references
//     does not point to the correct location
func (r *registry) DownloadBundle(imageName, outputDir string) error {
	return r.downloadBundleOrImage(imageName, outputDir, true)
}

// DownloadImage downloads an OCI image similarly to the `imgpkg pull -i` command
func (r *registry) DownloadImage(imageName, outputDir string) error {
	return r.downloadBundleOrImage(imageName, outputDir, false)
}

// CopyImageToTar downloads the image as tar file
// This is equivalent to `imgpkg copy --image <image> --to-tar <tar-file-path>` command
func (r *registry) CopyImageToTar(sourceImageName, destTarFile string) error {
	// Creating a dummy writer to capture the logs
	writerUI := ui.NewWriterUI(&writer{}, &writer{}, nil)

	copyOptions := cmd.NewCopyOptions(ui.NewWrappingConfUI(writerUI, nil))
	copyOptions.Concurrency = 3
	copyOptions.SignatureFlags = cmd.SignatureFlags{CopyCosignSignatures: true}
	isBundle, _ := bundle.NewBundle(sourceImageName, r.registry).IsBundle()
	if isBundle {
		copyOptions.BundleFlags = cmd.BundleFlags{Bundle: sourceImageName}
	} else {
		copyOptions.ImageFlags = cmd.ImageFlags{Image: sourceImageName}
	}
	copyOptions.TarFlags.TarDst = destTarFile

	if r.opts != nil {
		copyOptions.RegistryFlags = cmd.RegistryFlags{
			CACertPaths: r.opts.CACertPaths,
			VerifyCerts: r.opts.VerifyCerts,
			Insecure:    r.opts.Insecure,
			Anon:        r.opts.Anon,
		}
	}

	err := copyOptions.Run()
	if err != nil {
		return err
	}
	return nil
}

// CopyImageFromTar publishes the image to destination repository from specified tar file
// This is equivalent to `imgpkg copy --tar <file> --to-repo <dest-repo>` command
func (r *registry) CopyImageFromTar(sourceTarFile, destImageRepo string) error {
	// Creating a dummy writer to capture the logs
	writerUI := ui.NewWriterUI(&writer{}, &writer{}, nil)

	copyOptions := cmd.NewCopyOptions(ui.NewWrappingConfUI(writerUI, nil))
	copyOptions.Concurrency = 1
	copyOptions.TarFlags.TarSrc = sourceTarFile
	copyOptions.RepoDst = destImageRepo
	if r.opts != nil {
		copyOptions.RegistryFlags = cmd.RegistryFlags{
			CACertPaths: r.opts.CACertPaths,
			VerifyCerts: r.opts.VerifyCerts,
			Insecure:    r.opts.Insecure,
		}
	}
	err := copyOptions.Run()
	if err != nil {
		return err
	}
	return nil
}

func (r *registry) downloadBundleOrImage(imageName, outputDir string, isBundle bool) error {
	// Creating a dummy writer to capture the logs
	// currently this logs are not displayed or used directly
	var outputBuf, errorBuf bytes.Buffer
	writerUI := ui.NewWriterUI(&outputBuf, &errorBuf, nil)

	pullOptions := cmd.NewPullOptions(writerUI)
	pullOptions.OutputPath = outputDir
	if isBundle {
		pullOptions.BundleFlags = cmd.BundleFlags{Bundle: imageName}
	} else {
		pullOptions.ImageFlags = cmd.ImageFlags{Image: imageName}
	}
	if r.opts != nil {
		pullOptions.RegistryFlags = cmd.RegistryFlags{
			CACertPaths: r.opts.CACertPaths,
			VerifyCerts: r.opts.VerifyCerts,
			Insecure:    r.opts.Insecure,
			Anon:        r.opts.Anon,
		}
	}

	return pullOptions.Run()
}

// GetImageDigest gets the digest of an OCI image similarly to the `imgpkg tag resolve -i` command
func (r *registry) GetImageDigest(imageWithTag string) (string, string, error) {
	ref, err := regname.ParseReference(imageWithTag, regname.WeakValidation)
	if err != nil {
		return "", "", err
	}
	hash, err := r.registry.Digest(ref)
	if err != nil {
		return "", "", err
	}
	return hash.Algorithm, hash.Hex, nil
}

// PushImage publishes the image to the specified location
// This is equivalent to `imgpkg push -i <image> -f <filepath>`
func (r *registry) PushImage(imageWithTag string, filePaths []string) error {
	// Creating a dummy writer to capture the logs
	// currently this logs are not displayed or used directly
	var outputBuf, errorBuf bytes.Buffer
	writerUI := ui.NewWriterUI(&outputBuf, &errorBuf, nil)

	pushOptions := cmd.NewPushOptions(writerUI)
	pushOptions.ImageFlags = cmd.ImageFlags{Image: imageWithTag}
	pushOptions.FileFlags = cmd.FileFlags{Files: filePaths}
	if r.opts != nil {
		pushOptions.RegistryFlags = cmd.RegistryFlags{
			CACertPaths: r.opts.CACertPaths,
			VerifyCerts: r.opts.VerifyCerts,
			Insecure:    r.opts.Insecure,
		}
	}

	return pushOptions.Run()
}

// ResolveImage invokes `imgpkg tag resolve -i <image>` command
func (r *registry) ResolveImage(imageWithTag string) error {
	// Creating a dummy writer to capture the logs
	// currently this logs are not displayed or used directly
	var outputBuf, errorBuf bytes.Buffer
	writerUI := ui.NewWriterUI(&outputBuf, &errorBuf, nil)

	resolveOptions := cmd.NewTagResolveOptions(writerUI)
	resolveOptions.ImageFlags = cmd.ImageFlags{Image: imageWithTag}
	if r.opts != nil {
		resolveOptions.RegistryFlags = cmd.RegistryFlags{
			CACertPaths: r.opts.CACertPaths,
			VerifyCerts: r.opts.VerifyCerts,
			Insecure:    r.opts.Insecure,
			Anon:        r.opts.Anon,
		}
	}

	return resolveOptions.Run()
}
