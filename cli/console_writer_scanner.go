/*
Copyright 2024 Kasai Kou

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
)

type ProcessOutputDecoration struct{ Stdout, Stderr Decoration }

var ProcessOutputDecorations = []ProcessOutputDecoration{
	{
		Stdout: Decoration{Foreground: FG_BLUE, Bold: DC_BOLD},
		Stderr: Decoration{Foreground: FG_WHITE, Background: BG_BLUE, Bold: DC_BOLD},
	}, {
		Stdout: Decoration{Foreground: FG_YELLOW, Bold: DC_BOLD},
		Stderr: Decoration{Foreground: FG_BLACK, Background: BG_YELLOW, Bold: DC_BOLD},
	}, {
		Stdout: Decoration{Foreground: FG_CYAN, Bold: DC_BOLD},
		Stderr: Decoration{Foreground: FG_BLACK, Background: BG_CYAN, Bold: DC_BOLD},
	}, {
		Stdout: Decoration{Foreground: FG_MAGENTA, Bold: DC_BOLD},
		Stderr: Decoration{Foreground: FG_WHITE, Background: BG_MAGENTA, Bold: DC_BOLD},
	}, {
		Stdout: Decoration{Foreground: FG_GREEN, Bold: DC_BOLD},
		Stderr: Decoration{Foreground: FG_WHITE, Background: BG_GREEN, Bold: DC_BOLD},
	}, {
		Stdout: Decoration{Foreground: FG_RED, Bold: DC_BOLD},
		Stderr: Decoration{Foreground: FG_WHITE, Background: BG_RED, Bold: DC_BOLD},
	},
}

type ConsoleWriterScaner struct {
	dest            *ConsoleWriter
	labelDecoration Decoration
	kind            string
	label           string
}

var adjustLabelPrefix = regexp.MustCompile(`^[^a-zA-Z0-9]*[a-zA-Z0-9]+[^a-zA-Z0-9]?`)
var adjustLabelSuffix = regexp.MustCompile(`[^a-zA-Z0-9]?[a-zA-Z0-9]+[^a-zA-Z0-9]*$`)

func adjustLabel(label string) string {
	if len(label) <= RecordLabelWidthLimit {
		return label
	}

	wantWidth := RecordLabelWidthLimit - 3
	first := adjustLabelPrefix.FindStringIndex(label)
	last := adjustLabelSuffix.FindStringIndex(label)

	var prefixSize, suffixSize int

	if (first[0] == 0 && first[1] == len(label)) || last == nil {
		prefixSize = (wantWidth / 2) - 1
		suffixSize = wantWidth - prefixSize

	} else if suffixLength := last[1] - last[0]; suffixLength > wantWidth-5 {
		prefixSize = 5
		suffixSize = wantWidth - prefixSize
	} else if prefixLength := first[1] - first[0]; prefixLength > wantWidth-5 {
		suffixSize = 5
		prefixSize = wantWidth - suffixSize
	} else if prefixLength+suffixLength > wantWidth {
		suffixSize = suffixLength
		prefixSize = wantWidth - suffixSize
	} else {
		prefixSize = prefixLength
		suffixSize = wantWidth - prefixSize
	}

	label = label[:prefixSize] + "..." + label[len(label)-suffixSize:]
	return label
}

func (cw *ConsoleWriter) NewScanner(labelDecoration Decoration, kind string, label string) *ConsoleWriterScaner {
	scanner := &ConsoleWriterScaner{
		dest:            cw,
		labelDecoration: labelDecoration,
		kind:            kind,
		label:           adjustLabel(label),
	}

	return scanner
}

func (cws *ConsoleWriterScaner) Scan(reader io.Reader) {

	ch := cws.dest.chRecord

	decoration := Decoration{}
	scanner := bufio.NewScanner(reader)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if crIdx := bytes.IndexByte(data, '\r'); crIdx > -1 {
			if data[crIdx+1] == '\n' {
				return crIdx + 2, data[:crIdx+2], nil
			} else {
				return crIdx + 1, data[:crIdx+1], nil
			}
		} else if lfIdx := bytes.IndexByte(data, '\n'); lfIdx > -1 {
			return lfIdx + 1, data[:lfIdx+1], nil
		}

		if atEOF {
			return len(data), append(data, '\n'), nil
		}

		return 0, nil, nil
	})

	for scanner.Scan() {
		mode := RecordModeNA
		line := scanner.Bytes()
		decoration = decoration.Push(line)
		if len(line) > 0 && line[len(line)-1] == '\n' {
			mode = RecordModeLF
			line = line[:len(line)-1]
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			if mode == RecordModeCR {
				mode = RecordModeCR
			}
			line = line[:len(line)-1]
		}

		if mode == RecordModeNA {
			panic("invalid split")
		}

		ch <- ConsoleRecord{
			sender:          cws,
			RecordMode:      mode,
			LabelDecoration: cws.labelDecoration,
			Kind:            cws.kind,
			Label:           cws.label,
			Text:            string(line),
		}
	}
}
