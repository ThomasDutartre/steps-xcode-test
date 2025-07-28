package testaddon

import (
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
	IsCompilationFailure  bool
}

func (e exporter) CopyAndSaveMetadata(info AddonCopy) error {
	originalBundleName := info.TargetAddonBundleName
	info.TargetAddonBundleName = e.testAddon.ReplaceUnsupportedFilenameCharacters(info.TargetAddonBundleName)
	addonPerStepOutputDir := filepath.Join(info.TargetAddonPath, info.TargetAddonBundleName)

	// Log all the important info in debug mode
	logger := e.testAddon.GetLogger()
	logger.Debugf("CopyAndSaveMetadata called:")
	logger.Debugf("  - SourceTestOutputDir: '%s'", info.SourceTestOutputDir)
	logger.Debugf("  - TargetAddonPath: '%s'", info.TargetAddonPath)
	logger.Debugf("  - TargetAddonBundleName (original): '%s'", originalBundleName)
	logger.Debugf("  - TargetAddonBundleName (cleaned): '%s'", info.TargetAddonBundleName)
	logger.Debugf("  - addonPerStepOutputDir: '%s'", addonPerStepOutputDir)
	logger.Debugf("  - IsCompilationFailure: %v", info.IsCompilationFailure)

	// Only copy directory if source exists (for normal test results)
	if info.SourceTestOutputDir != "" {
		logger.Debugf("Copying existing test results from '%s'", info.SourceTestOutputDir)
		if err := e.testAddon.CopyDirectory(info.SourceTestOutputDir, addonPerStepOutputDir); err != nil {
			return err
		}
	} else {
		logger.Debugf("Creating directory for compilation failure (no source to copy)")
		// For compilation failures, just create the target directory
		if err := e.testAddon.CreateDirectory(addonPerStepOutputDir); err != nil {
			return err
		}
	}

	logger.Debugf("Saving bundle metadata to '%s'", addonPerStepOutputDir)
	if err := e.testAddon.SaveBundleMetadata(addonPerStepOutputDir, info.TargetAddonBundleName, info.IsCompilationFailure); err != nil {
		return err
	}

	logger.Debugf("CopyAndSaveMetadata completed successfully")
	return nil
}
