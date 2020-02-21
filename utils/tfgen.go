package utils

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/hcl/printer"
)

type TerraformFileConfig struct {
	Flags     *flag.FlagSet
	ListFlags []string
	MapFlags  []string

	PreLines  []string
	BodyLines []string
	PostLines []string

	BodyPrefix string
}

func wrapLongLines(text string, lineWidth int) []string {
	var ret []string
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return []string{text}
	}

	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			ret = append(ret, wrapped)
			wrapped = word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	ret = append(ret, wrapped)
	return ret
}

func printFlag(f *flag.Flag) {
	lines := wrapLongLines(f.Usage, 60)
	fname := fmt.Sprintf("-%s=", f.Name)

	fmt.Println("")
	for i, line := range lines {
		if i == 0 {
			if len(fname) > 20 {
				fmt.Printf("  %s\n", fname)
				fmt.Printf("  %-20s %s\n", "", line)
			} else {
				fmt.Printf("  %-20s %s\n", fname, line)
			}
		} else {
			fmt.Printf("  %-20s %s\n", "", line)
		}
	}
}

func ComposeTerraformFile(cfg *TerraformFileConfig) ([]byte, error) {
	return nil, nil
}

func (c *TerraformFileConfig) IsList(name string) bool {
	for _, n := range c.ListFlags {
		if n == name {
			return true
		}
	}
	return false
}

func (c *TerraformFileConfig) IsMap(name string) bool {
	for _, n := range c.ListFlags {
		if n == name {
			return true
		}
	}
	return false
}

func (c *TerraformFileConfig) PrintOptionHelp() {
	c.Flags.VisitAll(printFlag)
}

func (c *TerraformFileConfig) Generate() ([]byte, error) {
	var lines []string = c.BodyLines
	var errs []error

	listValues := make(map[string][]string)
	mapValues := make(map[string]map[string]string)

	c.Flags.Visit(func(f *flag.Flag) {
		// If that variable already exists in the body, remove it
		for i, l := range lines {
			if strings.Contains(l, f.Name) {
				copy(lines[i:], lines[i+1:]) // Shift a[i+1:] left one index.
				lines[len(lines)-1] = ""     // Erase last element (write zero value).
				lines = lines[:len(lines)-1] // Truncate slice.
				break
			}
		}

		if c.IsList(f.Name) {
			// If that's a list, add it on the list
			var vals []string
			if v, ok := listValues[f.Name]; ok {
				vals = v
			}

			vals = append(vals, fmt.Sprintf("%s", f.Value.String()))
			listValues[f.Name] = vals

		} else if c.IsList(f.Name) {
			// If that's a map, add it on the maps
			value := f.Value.String()
			kv := strings.Split(value, "=")
			if len(kv) < 2 {
				errs = append(errs, fmt.Errorf("Could not parse '%s': Expected key=value format", value))
				return
			}

			vals := make(map[string]string)
			if v, ok := mapValues[f.Name]; ok {
				vals = v
			}

			vals[kv[0]] = kv[1]
			mapValues[f.Name] = vals

		} else {
			// Otherwise append it to the list
			v, _ := json.Marshal(f.Value.String())
			lines = append(lines, fmt.Sprintf("%s = %s", f.Name, string(v)))
		}
	})

	// Then expand the lists
	for varName, list := range listValues {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("%s = [", varName))
		for _, item := range list {
			v, _ := json.Marshal(item)
			lines = append(lines, fmt.Sprintf("  %s,", v))
		}
		lines = append(lines, fmt.Sprintf("]"))
	}

	// Then expand the maps
	for varName, list := range mapValues {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("%s = {", varName))
		for key, item := range list {
			v, _ := json.Marshal(item)
			lines = append(lines, fmt.Sprintf("  %s = %s", key, v))
		}
		lines = append(lines, fmt.Sprintf("}"))
	}

	// Compose all lines
	allLines := append(c.PreLines, c.BodyLines...)
	allLines = append(allLines, lines...)
	allLines = append(allLines, c.PostLines...)

	content := []byte(strings.Join(allLines, "\n"))

	content, err := printer.Format(content)
	if err != nil {
		return nil, fmt.Errorf("Could not format output: %s", err.Error())
	}

	return content, nil
}
