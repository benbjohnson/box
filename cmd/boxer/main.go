package main

import (
	"flag"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/benbjohnson/boxer"
)

func main() {
	m := NewMain()
	if err := m.Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

// DefaultTickInterval is the time between ticks on the ticker.
const DefaultTickInterval = 1 * time.Second

// Main represents the program execution.
type Main struct {
	// The time between tick execution on the Ticker.
	TickInterval time.Duration

	// The function used to execute OS commands.
	Executor boxer.CommandExecutor

	// The logger passed to the ticker during execution.
	Logger *log.Logger

	closing chan struct{}
}

// NewMain returns a new instance of Main with default settings.
func NewMain() *Main {
	return &Main{
		TickInterval: DefaultTickInterval,
		Executor:     boxer.DefaultCommandExecutor,
		Logger:       log.New(os.Stderr, "", 0),

		closing: make(chan struct{}, 0),
	}
}

// Run excutes the program.
func (m *Main) Run(args []string) error {
	// Parse CLI arguments.
	configPath := flag.String("config", "", "config path")
	fs := flag.NewFlagSet("boxer", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Read configuration file.
	config, err := m.ReadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("read config: %s", err)
	}

	// Use a temp directory if no work directory is set.
	if config.WorkDir == "" {
		str, err := ioutil.TempDir("", "boxer-")
		if err != nil {
			return fmt.Errorf("temp dir: %s", err)
		}
		config.WorkDir = str
	}

	// Create a new ticker based on the config.
	ticker, err := NewTicker(config, m.Executor)
	if err != nil {
		return fmt.Errorf("cannot create ticker: %s", err)
	}

	// Notify user of the current settings.
	log.Printf("Boxer running with %d commands...", len(ticker.Commands))

	// Begin ticking.
	for {
		ticker.Tick()
		time.Sleep(m.TickInterval)
	}
}

// ReadConfig reads the configuration from a path.
// If no path is provided then the default path is used.
func (m *Main) ReadConfig(path string) (*Config, error) {
	// If no path is provided then use the default path.
	if path == "" {
		str, err := DefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("default config path: %s", err)
		}
		path = str
	}

	// Decode file into config.
	config := NewConfig()
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// DefaultConfigPath returns the default configuration path.
// The default path is the "boxer.conf" file in the user's home directory.
func DefaultConfigPath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, "boxer.conf"), nil
}

// NewTicker creates a new ticker from configuration.
func NewTicker(c *Config, exec boxer.CommandExecutor) (*boxer.Ticker, error) {
	t := boxer.NewTicker()

	if c.Wallpaper.Enabled {
		// Parse foreground color from config.
		foreground, err := ParseColor(c.Wallpaper.Foreground)
		if err != nil {
			return nil, fmt.Errorf("parse wallpaper foreground: %s", err)
		}

		// Parse backgroun color from config.
		background, err := ParseColor(c.Wallpaper.Background)
		if err != nil {
			return nil, fmt.Errorf("parse wallpaper background: %s", err)
		}

		// Create a wallpaper generator.
		generator := boxer.NewWallpaperGenerator(foreground, background)

		// Generate a new command.
		t.Commands = append(t.Commands, boxer.Command{
			Name:     "wallpaper",
			Step:     c.Wallpaper.Step.Duration,
			Interval: c.Wallpaper.Interval.Duration,
			Handler: boxer.NewWallpaperHandler(
				exec, boxer.DesktopSize, generator,
				filepath.Join(c.WorkDir, "wallpaper"),
			),
		})
	}

	return t, nil
}

// Config represnts the configuration file used to store command settings.
type Config struct {
	WorkDir string `toml:"work_dir"`

	Wallpaper struct {
		Enabled    bool     `toml:"enabled"`
		Step       Duration `toml:"step"`
		Interval   Duration `toml:"interval"`
		Foreground string   `toml:"foreground"`
		Background string   `toml:"background"`
	} `toml:"wallpaper"`
}

// NewConfig returns an instance of Config with default settings.
func NewConfig() *Config {
	var c Config
	c.Wallpaper.Enabled = false
	c.Wallpaper.Step = Duration{1 * time.Minute}
	c.Wallpaper.Interval = Duration{15 * time.Minute}
	c.Wallpaper.Foreground = "#9AC97C"
	c.Wallpaper.Background = "#534B4D"
	return &c
}

// Duration is used by the TOML config to parse duration values.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}

	d.Duration = v
	return nil
}

// ParseColor parses a hex color.
func ParseColor(s string) (color.RGBA, error) {
	m := regexp.MustCompile(`^#?([0-9a-fA-F]{2})([0-9a-fA-F]{2})([0-9a-fA-F]{2})$`).FindStringSubmatch(s)
	if m == nil {
		return color.RGBA{}, fmt.Errorf("cannot parse color: %q", s)
	}

	r, _ := strconv.ParseUint(m[1], 16, 8)
	g, _ := strconv.ParseUint(m[2], 16, 8)
	b, _ := strconv.ParseUint(m[3], 16, 8)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}, nil
}

func warn(v ...interface{})              { fmt.Fprintln(os.Stderr, v...) }
func warnf(msg string, v ...interface{}) { fmt.Fprintf(os.Stderr, msg+"\n", v...) }
