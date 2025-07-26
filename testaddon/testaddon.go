package testaddon

import "path/filepath"

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
	info.TargetAddonBundleName = e.testAddon.ReplaceUnsupportedFilenameCharacters(info.TargetAddonBundleName)
	addonPerStepOutputDir := filepath.Join(info.TargetAddonPath, info.TargetAddonBundleName)

	// Only copy directory if source exists (for normal test results)
	if info.SourceTestOutputDir != "" {
		if err := e.testAddon.CopyDirectory(info.SourceTestOutputDir, addonPerStepOutputDir); err != nil {
			return err
		}
	} else {
		// For compilation failures, just create the target directory
		if err := e.testAddon.CreateDirectory(addonPerStepOutputDir); err != nil {
			return err
		}
	}

	if err := e.testAddon.SaveBundleMetadata(addonPerStepOutputDir, info.TargetAddonBundleName); err != nil {
		return err
	}
	return nil
}
