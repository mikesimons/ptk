package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPtk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ptk Suite")
}
