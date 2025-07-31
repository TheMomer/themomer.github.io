// Copyright (c) 2025 TheMomer.
// Licensed under the MIT License.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/pflag"
	lua "github.com/yuin/gopher-lua"
)

// Setting the constants
const (
	Version string = "0.1.0 Alpha"
	//License = "MIT"
	//Creator = "TheMomer"
)

// Config structure
type Config struct {
	Modules      []Module // Module array
	Separator    string   // Separator
	Logo         string   // Logotip
	PaddingRight int      // Padding Right
	PaddingLeft  int      // Padding Left
	PaddingTop   int      // Padding Top
	Mode         string   // Display Mode
	Shell        []string // Shell for execCmd function
}

// Module structure
type Module struct {
	Text string
}

// Function for formatting text
func formatText(s string) string {
	// Colors
	colors := map[string]string{
		"black":         "\033[30m",
		"red":           "\033[31m",
		"green":         "\033[32m",
		"yellow":        "\033[33m",
		"blue":          "\033[34m",
		"magenta":       "\033[35m",
		"cyan":          "\033[36m",
		"white":         "\033[37m",
		"black_light":   "\033[90m",
		"red_light":     "\033[91m",
		"green_light":   "\033[92m",
		"yellow_light":  "\033[93m",
		"blue_light":    "\033[94m",
		"magenta_light": "\033[95m",
		"cyan_light":    "\033[96m",
		"white_light":   "\033[97m",
	}

	// Styles
	styles := map[string]string{
		"reset":            "\033[0m",
		"bold":             "\033[1m",
		"faint":            "\033[2m",
		"italic":           "\033[3m",
		"underline":        "\033[4m",
		"blink":            "\033[5m",
		"blink_fast":       "\033[6m",
		"reverse":          "\033[7m",
		"hidden":           "\033[8m",
		"strikethrough":    "\033[9m",
		"double_underline": "\033[21m",
		"overline":         "\033[53m",
	}

	// Return value
	re := regexp.MustCompile(`\{(\w+)\.([^}]+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		category, value := submatches[1], submatches[2]

		switch category {
		case "color":
			// Hex colors will work only if the terminal supports True Color
			// For example: Kitty, Windows Terminal,
			// Konsole, iTerm2, Alacritty
			if strings.HasPrefix(value, "#") {
				hex := strings.TrimPrefix(value, "#")
				if len(hex) != 6 {
					return match
				}

				r, _ := strconv.ParseUint(hex[0:2], 16, 8)
				g, _ := strconv.ParseUint(hex[2:4], 16, 8)
				b, _ := strconv.ParseUint(hex[4:6], 16, 8)

				return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
			} else if code, ok := colors[value]; ok {
				// Classic ANSI colors
				// Works in all terminals
				return code
			}

		case "style":
			if code, ok := styles[value]; ok {
				return code
			}
		}

		return match
	})
}

// Function for displaying an error, warning or information
func echoMsg(str string, detail interface{}, type_ string) {
	var prefix, message string

	switch type_ {
	// Error
	case "err":
		reason, ok := detail.(error)
		if !ok {
			fmt.Fprintf(os.Stderr, "expected error type for 'err', got %T\n", detail)
			return
		}
		prefix = fmt.Sprintf("{color.red_light}(!) %s:{style.reset}", str)
		message = reason.Error()
	// Warn
	case "warn":
		msg, ok := detail.(string)
		if !ok {
			fmt.Fprintf(os.Stderr, "expected string type for 'warn', got %T\n", detail)
			return
		}
		prefix = fmt.Sprintf("{color.yellow_light}/!\\ %s:{style.reset}", str)
		message = msg
	// Info
	case "info":
		msg, ok := detail.(string)
		if !ok {
			fmt.Fprintf(os.Stderr, "expected string type for 'info', got %T\n", detail)
			return
		}
		prefix = fmt.Sprintf("{color.blue_light}(i) %s:{style.reset}", str)
		message = msg
	// Unknown type
	default:
		fmt.Fprintf(os.Stderr, "unknown message type: %s\n", type_)
		return
	}

	// Display an error, warning or information
	fmt.Println(formatText(prefix))
	fmt.Fprintln(os.Stderr, message)
}

// Function to execute the command
func execCmd(cmd string, shell []string) (string, error) {
	var command *exec.Cmd

	args := append(shell[1:], cmd)
	command = exec.Command(shell[0], args...)

	// Execute command
	out, err := command.CombinedOutput()
	trimmedOut := strings.TrimSpace(string(out))

	// Format error with command output if exists
	if err != nil {
		return trimmedOut, fmt.Errorf("%s: %w", trimmedOut, err)
	}

	// Return clean output without errors
	return trimmedOut, nil
}

// Helper function for module registration
func RegisterModule(L *lua.LState, name string, methods map[string]lua.LGFunction, fields map[string]lua.LValue) {
	L.PreloadModule(name, func(L *lua.LState) int {
		module := L.NewTable()
		L.SetFuncs(module, methods)
		for key, value := range fields {
			L.SetField(module, key, value)
		}
		L.Push(module)
		return 1
	})
}

// Function for loading the config
func loadConfig(filename string) (*Config, error) {
	L := lua.NewState()
	defer L.Close()

	// Default settings
	cfg := &Config{
		PaddingRight: 2,
		PaddingLeft:  0,
		PaddingTop:   0,
		Mode:         "default",
		Shell:        []string{"/bin/sh", "-c"},
		Logo: `{color.green_light}__   __    {style.reset}
{color.green_light}\ \ / /__  {style.reset}
{color.green_light} \ V / _ \ {style.reset}
{color.green_light}  | | (_) |{style.reset}
{color.green_light}  |_|\___/ {style.reset}`,
	}

	// fields for module "yo" in Lua
	fields := map[string]lua.LValue{
		"version":          lua.LString(Version),
		"default_distance": lua.LNumber(2),
		"default_mode":     lua.LString("default"),
	}

	// Register yo module with simplified functions
	RegisterModule(L, "Yofetch", map[string]lua.LGFunction{
		"exec": func(L *lua.LState) int {
			cmd := L.CheckString(1)
			output, err := execCmd(cmd, cfg.Shell)

			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}

			L.Push(lua.LString(output))
			return 1
		},

		"logo": func(L *lua.LState) int {
			pathOrContent := L.CheckString(1)

			if data, err := os.ReadFile(pathOrContent); err == nil {
				cfg.Logo = string(data)
			} else {
				cfg.Logo = pathOrContent
			}

			return 0
		},

		"padding": func(L *lua.LState) int {
			paddingRight := L.CheckInt(1)
			paddingLeft := L.CheckInt(2)
			paddingTop := L.CheckInt(3)

			if paddingRight < 0 || paddingLeft < 0 || paddingTop < 0 {
				L.RaiseError("padding cannot be negative")
			}

			cfg.PaddingRight = paddingRight
			cfg.PaddingLeft = paddingLeft
			cfg.PaddingTop = paddingTop
			return 0
		},

		"mode": func(L *lua.LState) int {
			mode := L.CheckString(1)
			cfg.Mode = mode
			return 0
		},

		"print": func(L *lua.LState) int {
			text := L.CheckString(1)
			cfg.Modules = append(cfg.Modules, Module{
				Text: text,
			})
			return 0
		},

		"shell": func(L *lua.LState) int {
			table := L.CheckTable(1)
			var shellname []string

			table.ForEach(func(_, value lua.LValue) {
				if str, ok := value.(lua.LString); ok {
					shellname = append(shellname, string(str))
				}
			})

			cfg.Shell = shellname
			return 0
		},
	}, fields)

	// Execute the configuration file
	err := L.DoFile(filename)

	if err != nil {
		return nil, fmt.Errorf("Lua config error:\n%w", err)
	}

	return cfg, nil
}

// Function for building information
func buildInfo(cfg *Config) string {
	var info strings.Builder

	for _, module := range cfg.Modules {
		// Formatting the module Text
		textFormatted := formatText(module.Text)
		info.WriteString(textFormatted + "\n")
	}

	// Returning the information
	return info.String()
}

// Function for displaying the logo on the left and information on the right
func printLogoWithInfo(logo, info string, distance, paddingLeft, paddingTop int) {
	var ansiRegex = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)
	type Line struct {
		Original string // Original string with ANSI codes
		Clean    string // Clear string for width calculation
	}

	// Preparing the logo lines
	var logoLines []Line
	for i := 0; i < paddingTop; i++ {
		logoLines = append(logoLines, Line{
			Original: "",
			Clean:    "",
		})
	}
	for _, line := range strings.Split(strings.Trim(logo, "\n"), "\n") {
		clean := ansiRegex.ReplaceAllString(line, "")
		logoLines = append(logoLines, Line{
			Original: line,
			Clean:    clean,
		})
	}

	// Preparation of information lines
	var infoLines []Line
	for _, line := range strings.Split(strings.Trim(info, "\n"), "\n") {
		clean := ansiRegex.ReplaceAllString(line, "")
		infoLines = append(infoLines, Line{
			Original: line,
			Clean:    clean,
		})
	}

	// Calculating the maximum width of the logo
	maxLogoWidth := 0
	for _, line := range logoLines {
		if width := utf8.RuneCountInString(line.Clean); width > maxLogoWidth {
			maxLogoWidth = width
		}
	}

	// Line-by-line output
	maxLines := max(len(logoLines), len(infoLines))
	for i := 0; i < maxLines; i++ {
		var (
			logoLine Line
			infoLine Line
		)

		if i < len(logoLines) {
			logoLine = logoLines[i]
		}
		if i < len(infoLines) {
			infoLine = infoLines[i]
		}

		// Calculation of indentation between logo and information
		padding := maxLogoWidth - utf8.RuneCountInString(logoLine.Clean) + distance

		// Building string
		var output strings.Builder
		output.WriteString(strings.Repeat(" ", paddingLeft))
		output.WriteString(logoLine.Original)
		output.WriteString(strings.Repeat(" ", padding))
		output.WriteString(infoLine.Original)

		// Forced style reset and line break
		if !ansiRegex.MatchString(infoLine.Original) {
			output.WriteString("\033[0m")
		}
		output.WriteByte('\n')

		// Output
		fmt.Print(output.String())
	}
	os.Stdout.Sync()
}

// Main function
func main() {
	// Arguments
	config := pflag.StringP("config", "c", "config.lua", "path to config")

	// Parsing
	pflag.Parse()

	// Loading the config
	cfg, err := loadConfig(*config)
	if err != nil {
		echoMsg("Error loading config", err, "err")
		os.Exit(1)
	}

	// Formatting the logo
	logoFormatted := formatText(string(cfg.Logo))

	// Building information
	info := buildInfo(cfg)

	// Display the logo with information
	switch {
	case cfg.Logo == "":
		// If there is no logo, write just information in any mode
		fmt.Println(info)
	case cfg.Mode == "default":
		// Mode default with logo: write logo on the left and information on the right
		printLogoWithInfo(logoFormatted, info, cfg.PaddingRight, cfg.PaddingLeft, cfg.PaddingTop)
	case cfg.Mode == "vertical":
		// Mode vertical with logo: write logo first and then information
		fmt.Println(logoFormatted)
		fmt.Println(info)
	default:
		// The “default” mode will be used
		// if no mode is specified or “set_mode” has a different value
		printLogoWithInfo(logoFormatted, info, cfg.PaddingRight, cfg.PaddingLeft, cfg.PaddingTop)
	}
}
