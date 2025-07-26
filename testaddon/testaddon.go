package testaddon

import (
	"fmt"
	"path/filepath"
)

// Exporter ...
type Exporter interface {
	CopyAndSaveMetadata(info AddonCopy) error
}

type exporter struct {
	testAddon TestAddon
}

// NewExporter ...
func NewExporter(testAddon TestAddon) Exporter {
	return &exporter{
		testAddon: testAddon,
	}
}

// AddonCopy ...
type AddonCopy struct {
	SourceTestOutputDir   string
	TargetAddonPath       string
	TargetAddonBundleName string
}

func (e exporter) CopyAndSaveMetadata(info AddonCopy) error {
	originalBundleName := info.TargetAddonBundleName
	info.TargetAddonBundleName = e.testAddon.ReplaceUnsupportedFilenameCharacters(info.TargetAddonBundleName)
	addonPerStepOutputDir := filepath.Join(info.TargetAddonPath, info.TargetAddonBundleName)

	// Log all the important info
	fmt.Printf("DEBUG: CopyAndSaveMetadata called:\n")
	fmt.Printf("  - SourceTestOutputDir: '%s'\n", info.SourceTestOutputDir)
	fmt.Printf("  - TargetAddonPath: '%s'\n", info.TargetAddonPath)
	fmt.Printf("  - Original bundle name: '%s'\n", originalBundleName)
	fmt.Printf("  - Sanitized bundle name: '%s'\n", info.TargetAddonBundleName)
	fmt.Printf("  - Final output dir: '%s'\n", addonPerStepOutputDir)

	// Only copy directory if source exists (for normal test results)
	if info.SourceTestOutputDir != "" {
		fmt.Printf("DEBUG: Copying existing test results from '%s'\n", info.SourceTestOutputDir)
		if err := e.testAddon.CopyDirectory(info.SourceTestOutputDir, addonPerStepOutputDir); err != nil {
			return err
		}
	} else {
		fmt.Printf("DEBUG: Creating directory for compilation failure (no source to copy)\n")
		// For compilation failures, just create the target directory
		if err := e.testAddon.CreateDirectory(addonPerStepOutputDir); err != nil {
			return err
		}
	}

	fmt.Printf("DEBUG: Saving bundle metadata to '%s'\n", addonPerStepOutputDir)
	if err := e.testAddon.SaveBundleMetadata(addonPerStepOutputDir, info.TargetAddonBundleName); err != nil {
		return err
	}
	
	fmt.Printf("DEBUG: CopyAndSaveMetadata completed successfully\n")
	return nil
}
