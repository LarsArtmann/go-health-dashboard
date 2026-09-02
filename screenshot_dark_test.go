package dashboard_test

import (
	"os"
	"testing"
)

// TestCaptureREADME_ScreenshotDark renders the dashboard in dark mode:
//
//	SCREENSHOT_OUTPUT_DARK=docs/screenshot-dark.png \
//	GO_HEALTH_DASHBOARD_CHROME=/path/to/chromium \
//	go test -run TestCaptureREADME_ScreenshotDark -v .
func TestCaptureREADME_ScreenshotDark(t *testing.T) {
	t.Parallel()

	captureThemeScreenshot(t, "SCREENSHOT_OUTPUT_DARK", "dark", os.Getenv("SCREENSHOT_OUTPUT_DARK"))
}
