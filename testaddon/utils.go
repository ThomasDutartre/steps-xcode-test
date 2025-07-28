package testaddon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

type TestAddon interface {
	ReplaceUnsupportedFilenameCharacters(s string) string
	CopyDirectory(sourceBundle string, targetDir string) error
	CreateDirectory(targetDir string) error
	SaveBundleMetadata(outputDir string, bundleName string, isCompilationFailure bool) error
	GetLogger() log.Logger
}

type testAddon struct {
	logger log.Logger
}

func NewTestAddon(logger log.Logger) TestAddon {
	return &testAddon{
		logger: logger,
	}
}

func (t testAddon) GetLogger() log.Logger {
	return t.logger
}

// ReplaceUnsupportedFilenameCharacters Replaces characters '/' and ':', which are unsupported in filnenames on macOS
func (t testAddon) ReplaceUnsupportedFilenameCharacters(s string) string {
	s = strings.Replace(s, "/", "-", -1)
	s = strings.Replace(s, ":", "-", -1)
	return s
}

func (t testAddon) CreateDirectory(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory (%s): %w", targetDir, err)
	}
	return nil
}

func (t testAddon) CopyDirectory(sourceBundle string, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory (%s): %w", targetDir, err)
	}

	// the leading `/` means to copy not the content but the whole dir
	// -a means a better recursive, with symlinks handling and everything
	cmd := command.NewFactory(env.NewRepository()).Create("cp", []string{"-a", sourceBundle, targetDir + "/"}, nil)
	//cmd := command.New("cp", "-a", sourceBundle, targetDir+"/")
	// TODO: migrate log
	t.logger.Donef("$ %s", cmd.PrintableCommandArgs())
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("copy failed: %w, output: %s", err, out)
	}

	return nil
}

func (t testAddon) SaveBundleMetadata(outputDir string, bundleName string, isCompilationFailure bool) error {
	t.logger.Debugf("SaveBundleMetadata called:")
	t.logger.Debugf("  - outputDir: '%s'", outputDir)
	t.logger.Debugf("  - bundleName: '%s'", bundleName)
	t.logger.Debugf("  - isCompilationFailure: %v", isCompilationFailure)

	// Save test bundle metadata with simple format (same as original)
	type testBundle struct {
		BundleName string `json:"test-name"`
	}

	bundle := testBundle{
		BundleName: bundleName,
	}

	bytes, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("could not encode metadata: %w", err)
	}

	testInfoPath := filepath.Join(outputDir, "test-info.json")
	t.logger.Debugf("Writing test-info.json to '%s'", testInfoPath)
	t.logger.Debugf("test-info.json content: %s", string(bytes))

	if err = os.WriteFile(testInfoPath, bytes, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// If this is a compilation failure, create a fake test result file
	// This will make GitHub Checks display it as a failed test
	if isCompilationFailure {
		t.logger.Debugf("This is a compilation failure, creating fake test result")
		if err := t.createFakeTestResult(outputDir); err != nil {
			return fmt.Errorf("failed to create fake test result: %w", err)
		}
	} else {
		t.logger.Debugf("This is not a compilation failure, no fake test result needed")
	}

	t.logger.Debugf("SaveBundleMetadata completed successfully")
	return nil
}

func (t testAddon) createFakeTestResult(outputDir string) error {
	t.logger.Debugf("createFakeTestResult called with outputDir: '%s'", outputDir)

	// Create a fake xcresult directory structure for compilation failures
	// This makes GitHub Checks think there was a test that failed
	resultDir := filepath.Join(outputDir, "result")
	fakeXcresultDir := filepath.Join(resultDir, "CompilationTest.xcresult")

	t.logger.Debugf("Creating fake xcresult directory: '%s'", fakeXcresultDir)

	if err := os.MkdirAll(fakeXcresultDir, 0700); err != nil {
		return fmt.Errorf("failed to create fake xcresult directory: %w", err)
	}

	// Create a minimal Info.plist to make it look like a real xcresult
	infoPlist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>FormatVersion</key>
	<string>3.0</string>
	<key>TestFailureIsExpected</key>
	<false/>
</dict>
</plist>`

	infoPlistPath := filepath.Join(fakeXcresultDir, "Info.plist")
	t.logger.Debugf("Writing Info.plist to '%s'", infoPlistPath)

	if err := os.WriteFile(infoPlistPath, []byte(infoPlist), 0600); err != nil {
		return fmt.Errorf("failed to create Info.plist: %w", err)
	}

	t.logger.Infof("Created fake test result for compilation failure at %s", fakeXcresultDir)
	t.logger.Debugf("createFakeTestResult completed successfully")
	return nil
}
