package main

import (
	"testing"

	. "github.com/mikesimons/ptk"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPtk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ptk Suite")
}

var _ = Describe("Ptk", func() {

})
