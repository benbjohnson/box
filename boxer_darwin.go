package boxer

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// OSAScriptPath is the path to the "osascript" binary.
const OSAScriptPath = `/usr/bin/osascript`

// NewWallpaperHandler returns a handler for visualizing steps with the desktop wallpaper.
func NewWallpaperHandler(exec CommandExecutor, sizer DesktopSizer, generator WallpaperGenerator, path string) Handler {
	return func(i, n int) error {
		// Retrieve desktop size.
		w, h, err := sizer(exec)
		if err != nil {
			return fmt.Errorf("desktop size: %s", err)
		}

		// Generate wallpaper if it doesn't exist.
		// The wallpaper is saved to a common location format so we can tell if
		// the desktop size changes and recompute a wallpaper on the fly.
		imgpath := filepath.Join(path, fmt.Sprintf("wallpaper_%04d_%04d_%02d_%02d.png", w, h, i, n))
		if _, err := os.Stat(imgpath); os.IsNotExist(err) {
			if err := generator(imgpath, w, h, float64(i)/float64(n)); err != nil {
				return fmt.Errorf("generate wallpaper: %s", err)
			}
		}

		// Execute AppleScript to update the current background.
		src := fmt.Sprintf(strings.TrimSpace(setWallpaperScript), imgpath)
		if b, err := exec(OSAScriptPath, nil, strings.NewReader(src)); err != nil {
			return fmt.Errorf("exec: %s", b)
		}
		return nil
	}
}

const setWallpaperScript = `
tell application "Finder"
  set desktop picture to POSIX file "%s"
end tell
`

// WallpaperGenerator generates a wallpaper at the given path.
type WallpaperGenerator func(path string, w, h int, pct float64) error

// GenerateWallpaper generates a PNG wallpaper with a given size and color.
// The wallpaper will draw the foreground covering pct percent of the image.
func NewWallpaperGenerator(foreground, background color.RGBA) WallpaperGenerator {
	return func(path string, w, h int, pct float64) error {
		// Ensure the parent directory exists.
		if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
			return fmt.Errorf("mkdir: %s", err)
		}

		// Create image with the foreground color covering a percentage of the background.
		m := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(m, m.Bounds(), &image.Uniform{background}, image.ZP, draw.Over)
		draw.Draw(m, image.Rect(0, 0, w, int(float64(h)*pct)), &image.Uniform{foreground}, image.Point{X: 0, Y: int(float64(h) * (1.0 - pct))}, draw.Over)

		// Open output file.
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		// Encode to file.
		if err := png.Encode(f, m); err != nil {
			return fmt.Errorf("png encode: %s", err)
		}

		return nil
	}
}

// DesktopSizer returns the size of the desktop screen.
type DesktopSizer func(exec CommandExecutor) (w, h int, err error)

// DesktopSize returns the size of the desktop screen.
func DesktopSize(exec CommandExecutor) (w, h int, err error) {
	if b, err := exec(OSAScriptPath, nil, strings.NewReader(strings.TrimSpace(desktopSizeScript))); err != nil {
		return 0, 0, fmt.Errorf("exec: %s", b)
	} else if m := regexp.MustCompile(`^\d+, \d+, (\d+), (\d+)`).FindStringSubmatch(string(b)); m == nil {
		return 0, 0, fmt.Errorf("unexpected exec output: %s", b)
	} else {
		w, _ = strconv.Atoi(m[1])
		h, _ = strconv.Atoi(m[2])
		return w, h, err
	}
}

const desktopSizeScript = `
tell application "Finder"
  get bounds of window of desktop
end tell
`

// NewMenuBarHandler returns a handler for flashing the menu bar.
func NewMenuBarHandler(exec CommandExecutor) Handler {
	return func(i, n int) error {
		// Flash the menu bar on the first step.
		if i == 0 {
			if b, err := exec(OSAScriptPath, nil, strings.NewReader(strings.TrimSpace(flashDarkModeScript))); err != nil {
				return fmt.Errorf("exec flash: %s", b)
			}
			return nil
		}

		// For other steps set dark mode on and off every other step.
		darkMode := (i%2 == 1)
		src := fmt.Sprintf(strings.TrimSpace(setDarkModeScript), darkMode)
		if b, err := exec(OSAScriptPath, nil, strings.NewReader(src)); err != nil {
			return fmt.Errorf("exec set dark mode: %s", b)
		}
		return nil
	}
}

const flashDarkModeScript = `
tell application "System Events"
  tell appearance preferences
    repeat 5 times
      set dark mode to true
      delay 1
      set dark mode to false
      delay 1
    end repeat
  end tell
end tell
`

const setDarkModeScript = `
tell application "System Events"
  tell appearance preferences
    set dark mode to %v
  end tell
end tell
`

// NewAnnouncementHandler returns a handler for announcing the current time.
func NewAnnouncementHandler(exec CommandExecutor) Handler {
	return func(i, n int) error {
		src := fmt.Sprintf(displayNotificationScript, time.Now().Format("3:04pm"))
		if b, err := exec(OSAScriptPath, nil, strings.NewReader(src)); err != nil {
			return fmt.Errorf("exec display notification: %s", b)
		}
		return nil
	}
}

const displayNotificationScript = `display notification %q with title "Boxer"`
