// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
)

func TestInventorySuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Builder Inventory Suite")
}

var _ = Describe("Unit tests for inventory init", func() {
	fakeImgpkgWrapper := &fakes.ImageOperationsImpl{}
	iip := InventoryInitOptions{
		Repository:          "test-repo.com",
		InventoryImageTag:   "latest",
		ImageOperationsImpl: fakeImgpkgWrapper,
	}

	var _ = Context("tests for the inventory init function", func() {

		var _ = It("when everything works as expected without error", func() {
			iip.Override = false
			fakeImgpkgWrapper.ResolveImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.PushImageReturns(nil)

			err := iip.InitializeInventory()
			Expect(err).NotTo(HaveOccurred())
		})

		var _ = It("when override is false but the image already exist on the repository", func() {
			iip.Override = false
			fakeImgpkgWrapper.ResolveImageReturns(nil)

			err := iip.InitializeInventory()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("image already exists on the repository. Use `--override` flag to override the content"))
		})

		var _ = It("when override is false and the image doesn't already exist on the repository but push fails", func() {
			iip.Override = false
			fakeImgpkgWrapper.ResolveImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.PushImageReturns(errors.New("unable to push image"))

			err := iip.InitializeInventory()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to push image"))
			Expect(err.Error()).To(ContainSubstring("error while publishing database to the repository as image"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s:%s", iip.Repository, helpers.PluginInventoryDBImageName, iip.InventoryImageTag)))
		})
	})
})
