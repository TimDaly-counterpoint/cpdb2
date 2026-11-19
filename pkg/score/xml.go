package score

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// makeUTF8Reader returns a reader that decodes input encoding into UTF-8.
func makeUTF8Reader(r io.Reader) (io.Reader, error) {
	br := bufio.NewReader(r)
	header, err := br.Peek(512)
	if err != nil && err != io.EOF {
		return nil, err
	}

	headerStr := strings.ToLower(string(header))
	switch {
	case strings.Contains(headerStr, "encoding=\"utf-16\"") || strings.Contains(headerStr, "encoding='utf-16'"):
		bomPolicy := unicode.UseBOM
		var endian unicode.Endianness = unicode.LittleEndian
		if len(header) >= 2 && header[0] == 0xFE && header[1] == 0xFF {
			endian = unicode.BigEndian
		}
		decoder := unicode.UTF16(endian, bomPolicy).NewDecoder()
		return transform.NewReader(br, decoder), nil

	case strings.Contains(headerStr, "iso-8859-1") || strings.Contains(headerStr, "latin1"):
		return transform.NewReader(br, charmap.ISO8859_1.NewDecoder()), nil

	case strings.Contains(headerStr, "windows-1252"):
		return transform.NewReader(br, charmap.Windows1252.NewDecoder()), nil

	default:
		return br, nil
	}
}

// LoadMusicXML reads and parses a MusicXML score file.
func LoadMusicXML(path string) (*Score, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	utf8Reader, err := makeUTF8Reader(file)
	if err != nil {
		return nil, fmt.Errorf("detect encoding: %w", err)
	}

	raw, err := io.ReadAll(utf8Reader)
	if err != nil {
		return nil, fmt.Errorf("read XML file: %w", err)
	}

	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "iso-8859-1", "latin1":
			return charmap.ISO8859_1.NewDecoder().Reader(input), nil
		case "windows-1252":
			return charmap.Windows1252.NewDecoder().Reader(input), nil
		case "utf-16", "utf-16le", "utf-16be":
			return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Reader(input), nil
		default:
			return input, nil
		}
	}

	var score Score
	if err := decoder.Decode(&score); err != nil {
		return nil, fmt.Errorf("decode MusicXML: %w", err)
	}

	return &score, nil
}
