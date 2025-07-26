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
	SaveBundleMetadata(outputDir string, bundleName string) error
}

type testAddon struct {
	logger log.Logger
}

func NewTestAddon(logger log.Logger) TestAddon {
	return &testAddon{
		logger: logger,
	}
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

func (t testAddon) SaveBundleMetadata(outputDir string, bundleName string) error {
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
	if err = os.WriteFile(filepath.Join(outputDir, "test-info.json"), bytes, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// If this is a compilation failure, create a fake test result file
	// This will make GitHub Checks display it as a failed test
	if strings.Contains(bundleName, "compilation-failure") {
		if err := t.createFakeTestResult(outputDir); err != nil {
			return fmt.Errorf("failed to create fake test result: %w", err)
		}
	}

	return nil
}

func (t testAddon) createFakeTestResult(outputDir string) error {
	// Create a fake xcresult directory structure for compilation failures
	// This makes GitHub Checks think there was a test that failed
	resultDir := filepath.Join(outputDir, "result")
	fakeXcresultDir := filepath.Join(resultDir, "CompilationTest.xcresult")

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

	if err := os.WriteFile(filepath.Join(fakeXcresultDir, "Info.plist"), []byte(infoPlist), 0600); err != nil {
		return fmt.Errorf("failed to create Info.plist: %w", err)
	}

	t.logger.Infof("Created fake test result for compilation failure at %s", fakeXcresultDir)
	return nil
}
