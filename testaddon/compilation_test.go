package testaddon

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GivenCompilationFailure_WhenExport_ThenCreatesFakeTestResult(t *testing.T) {
	// Given
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")
	bundleName := "MyScheme-compilation-failure"

	exporter := NewExporter(NewTestAddon(log.NewLogger()))

	// When
	err := exporter.CopyAndSaveMetadata(AddonCopy{
		SourceTestOutputDir:   "", // Empty for compilation failure
		TargetAddonPath:       outputDir,
		TargetAddonBundleName: bundleName,
		IsCompilationFailure:  true,
	})

	// Then
	assert.NoError(t, err)

	// Check that the basic metadata file is created with simple format
	jsonPath := filepath.Join(outputDir, bundleName, "test-info.json")
	require.FileExists(t, jsonPath)

	// Verify JSON content has the simple format (same as normal tests)
	jsonFile, err := os.Open(jsonPath)
	require.NoError(t, err)
	defer jsonFile.Close()

	bytes, err := io.ReadAll(jsonFile)
	require.NoError(t, err)

	type testBundle struct {
		BundleName string `json:"test-name"`
	}
	var bundle testBundle
	err = json.Unmarshal(bytes, &bundle)
	require.NoError(t, err)

	assert.Equal(t, bundleName, bundle.BundleName)

	// Check that fake test result structure is created
	fakeXcresultPath := filepath.Join(outputDir, bundleName, "result", "CompilationTest.xcresult")
	assert.DirExists(t, fakeXcresultPath)

	// Check that Info.plist exists in the fake xcresult
	infoPlistPath := filepath.Join(fakeXcresultPath, "Info.plist")
	assert.FileExists(t, infoPlistPath)
}

func Test_GivenNormalTest_WhenExport_ThenNoFakeTestResult(t *testing.T) {
	// Given
	resultDir, outputDir := prepareArtifacts(t)
	bundleName := "NormalTest" // Not containing "compilation-failure"

	exporter := NewExporter(NewTestAddon(log.NewLogger()))

	// When
	err := exporter.CopyAndSaveMetadata(AddonCopy{
		SourceTestOutputDir:   resultDir,
		TargetAddonPath:       outputDir,
		TargetAddonBundleName: bundleName,
		IsCompilationFailure:  false,
	})

	// Then
	assert.NoError(t, err)

	// Check that no fake test result is created for normal tests
	fakeXcresultPath := filepath.Join(outputDir, bundleName, "result", "CompilationTest.xcresult")
	assert.NoDirExists(t, fakeXcresultPath)

	// But the normal structure should exist
	normalXcresultPath := filepath.Join(outputDir, bundleName, "result", "test.xcresult")
	assert.FileExists(t, normalXcresultPath)
}
