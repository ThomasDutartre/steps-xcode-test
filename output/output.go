package output

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise/configs"
	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/ziputil"
	"github.com/bitrise-steplib/steps-xcode-test/testaddon"
)

// Exporter ...
type Exporter interface {
	ExportXCResultBundle(deployDir, xcResultPath, scheme string)
	ExportTestRunResult(failed bool)
	ExportCompilationFailure(scheme string, errorMessage string) error
	ExportXcodebuildBuildLog(deployDir, xcodebuildBuildLog string) error
	ExportXcodebuildTestLog(deployDir, xcodebuildTestLog string) error
	ExportSimulatorDiagnostics(deployDir, pth, name string) error
}

type exporter struct {
	envRepository     env.Repository
	logger            log.Logger
	outputExporter    export.Exporter
	testAddonExporter testaddon.Exporter
}

// NewExporter ...
func NewExporter(envRepository env.Repository, logger log.Logger, outputExporter export.Exporter, testAddonExporter testaddon.Exporter) Exporter {
	return &exporter{
		envRepository:     envRepository,
		logger:            logger,
		outputExporter:    outputExporter,
		testAddonExporter: testAddonExporter,
	}
}

func (e exporter) ExportTestRunResult(failed bool) {
	status := "succeeded"
	if failed {
		status = "failed"
	}
	if err := e.envRepository.Set("BITRISE_XCODE_TEST_RESULT", status); err != nil {
		e.logger.Warnf("Failed to export: BITRISE_XCODE_TEST_RESULT: %s", err)
	}
}

func (e exporter) ExportXCResultBundle(deployDir, xcResultPath, scheme string) {
	// export xcresult bundle
	if err := e.envRepository.Set("BITRISE_XCRESULT_PATH", xcResultPath); err != nil {
		e.logger.Warnf("Failed to export: BITRISE_XCRESULT_PATH: %s", err)
	}

	xcresultZipPath := filepath.Join(deployDir, filepath.Base(xcResultPath)+".zip")
	if err := e.outputExporter.ExportOutputFilesZip("BITRISE_XCRESULT_ZIP_PATH", []string{xcResultPath}, xcresultZipPath); err != nil {
		e.logger.Warnf("Failed to export: BITRISE_XCRESULT_ZIP_PATH: %s", err)
	}

	// export xcresult for the testing addon
	if addonResultPath := e.envRepository.Get(configs.BitrisePerStepTestResultDirEnvKey); len(addonResultPath) > 0 {
		e.logger.Println()
		e.logger.Infof("Exporting test results")

		if err := e.testAddonExporter.CopyAndSaveMetadata(testaddon.AddonCopy{
			SourceTestOutputDir:   xcResultPath,
			TargetAddonPath:       addonResultPath,
			TargetAddonBundleName: scheme,
		}); err != nil {
			e.logger.Warnf("Failed to export test results: %s", err)
		}
	}
}

func (e exporter) ExportCompilationFailure(scheme string, errorMessage string) error {
	e.logger.Infof("DEBUG: ExportCompilationFailure called - scheme: '%s', errorMessage: '%s'", scheme, errorMessage)

	// Export failed test result
	e.ExportTestRunResult(true)

	// Create test metadata for GitHub Checks even for compilation failures
	addonResultPath := e.envRepository.Get(configs.BitrisePerStepTestResultDirEnvKey)
	e.logger.Infof("DEBUG: BITRISE_TEST_RESULT_DIR = '%s'", addonResultPath)

	if len(addonResultPath) > 0 {
		e.logger.Println()
		e.logger.Infof("Exporting compilation failure as test result for GitHub Checks")

		bundleName := scheme + "-compilation-failure"
		e.logger.Infof("DEBUG: Creating fake test bundle with name: '%s'", bundleName)

		// Create a fake test result using the simple format (same as normal tests)
		// This will make GitHub Checks display the compilation failure as a failed test
		if err := e.testAddonExporter.CopyAndSaveMetadata(testaddon.AddonCopy{
			SourceTestOutputDir:   "", // Empty since we don't have xcresult for compilation failures
			TargetAddonPath:       addonResultPath,
			TargetAddonBundleName: bundleName,
		}); err != nil {
			e.logger.Errorf("DEBUG: Failed to call CopyAndSaveMetadata: %s", err)
			return fmt.Errorf("failed to export compilation failure metadata: %w", err)
		}

		e.logger.Infof("DEBUG: Successfully exported compilation failure metadata")
	} else {
		e.logger.Warnf("DEBUG: BITRISE_TEST_RESULT_DIR is empty, cannot export compilation failure")
	}

	return nil
}

func (e exporter) ExportXcodebuildBuildLog(deployDir, xcodebuildBuildLog string) error {
	pth, err := saveRawOutputToLogFile(xcodebuildBuildLog)
	if err != nil {
		e.logger.Warnf("Failed to save the Raw Output, err: %s", err)
	}

	deployPth := filepath.Join(deployDir, "xcodebuild_build.log")
	if err := command.CopyFile(pth, deployPth); err != nil {
		return fmt.Errorf("failed to copy xcodebuild output log file from (%s) to (%s): %w", pth, deployPth, err)
	}

	if err := e.envRepository.Set("BITRISE_XCODEBUILD_BUILD_LOG_PATH", deployPth); err != nil {
		e.logger.Warnf("Failed to export: BITRISE_XCODEBUILD_BUILD_LOG_PATH: %s", err)
	}

	return nil
}

func (e exporter) ExportXcodebuildTestLog(deployDir, xcodebuildTestLog string) error {
	pth, err := saveRawOutputToLogFile(xcodebuildTestLog)
	if err != nil {
		e.logger.Warnf("Failed to save the Raw Output: %s", err)
	}

	deployPth := filepath.Join(deployDir, "xcodebuild_test.log")
	if err := command.CopyFile(pth, deployPth); err != nil {
		return fmt.Errorf("failed to copy xcodebuild output log file from (%s) to (%s): %w", pth, deployPth, err)
	}

	if err := e.envRepository.Set("BITRISE_XCODEBUILD_TEST_LOG_PATH", deployPth); err != nil {
		e.logger.Warnf("Failed to export: BITRISE_XCODEBUILD_TEST_LOG_PATH: %s", err)
	}

	return nil
}

func (e exporter) ExportSimulatorDiagnostics(deployDir, pth, name string) error {
	outputPath := filepath.Join(deployDir, name)
	if err := ziputil.ZipDir(pth, outputPath, true); err != nil {
		return fmt.Errorf("failed to compress simulator diagnostics result: %w", err)
	}

	return nil
}
