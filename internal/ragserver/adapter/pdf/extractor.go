package pdf

import (
	"bytes"
	"fmt"
	"io"
	"slices"

	"seehuhn.de/go/geom/matrix"

	"seehuhn.de/go/postscript/cid"
	"seehuhn.de/go/postscript/type1/names"

	"seehuhn.de/go/sfnt"
	"seehuhn.de/go/sfnt/glyf"

	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/font/dict"
	"seehuhn.de/go/pdf/pagetree"
	"seehuhn.de/go/pdf/reader"
)

type extractor struct {
	pageMin, pageMax     int
	xRangeMin, xRangeMax float64
	showPageNumbers      bool
}

func (e *extractor) extractText(data io.ReadSeeker) ([]*bytes.Buffer, int, error) {
	r, err := pdf.NewReader(data, nil)
	if err != nil {
		return nil, 0, err
	}

	numPages, err := pagetree.NumPages(r)
	if err != nil {
		return nil, 0, err
	}

	startPage := e.pageMin
	endPage := e.pageMax
	if endPage > numPages {
		endPage = numPages
	}

	// -----------------------------------------------------------------------

	extraTextCache := make(map[font.Embedded]map[cid.CID]string)
	spaceWidth := make(map[font.Embedded]float64)

	var w *bytes.Buffer
	buffers := make([]*bytes.Buffer, 0, endPage-startPage+1)

	contents := reader.New(r, nil)
	contents.TextEvent = func(op reader.TextEvent, arg float64) {
		switch op {
		case reader.TextEventSpace:
			w0, ok := spaceWidth[contents.TextFont]
			if !ok {
				w0 = getSpaceWidth(contents.TextFont)
				spaceWidth[contents.TextFont] = w0
			}

			if arg > 0.3*w0 {
				fmt.Fprint(w, " ")
			}
		case reader.TextEventNL:
			fmt.Fprintln(w)
		case reader.TextEventMove:
			fmt.Fprintln(w)
		}
	}
	contents.Character = func(cid cid.CID, text string) error {
		if text == "" {
			F := contents.TextFont
			m, ok := extraTextCache[F]
			if !ok {
				m = getExtraMapping(r, contents.TextFont)
				extraTextCache[F] = m
			}
			text = m[cid]
		}

		// xUser, yUser := contents.GetTextPositionUser()

		xDev, _ := contents.GetTextPositionDevice()
		if xDev >= e.xRangeMin && xDev < e.xRangeMax {
			fmt.Fprint(w, text)
		}
		return nil
	}

	// -----------------------------------------------------------------------

	for pageNo := startPage; pageNo <= endPage; pageNo++ {
		_, pageDict, err := pagetree.GetPage(r, pageNo-1)
		if err != nil {
			return nil, 0, err
		}

		w = bytes.NewBuffer(nil)
		buffers = append(buffers, w)

		if e.showPageNumbers {
			fmt.Fprintln(w, "Page", pageNo)
			fmt.Fprintln(w)
		}

		err = contents.ParsePage(pageDict, matrix.Identity)
		if err != nil {
			return nil, 0, fmt.Errorf("error parsing page %d: %w", pageNo, err)
		}

		fmt.Fprintln(w)
	}

	return buffers, numPages, nil
}

func getSpaceWidth(F font.Embedded) float64 {
	Fe, ok := F.(font.FromFile)
	if !ok {
		return 280
	}

	d := Fe.GetDict()
	if d == nil {
		return 0
	}

	return spaceWidthHeuristic(d)
}

func getExtraMapping(r pdf.Getter, F font.Embedded) map[cid.CID]string {
	Fe, ok := F.(font.FromFile)
	if !ok {
		return nil
	}

	d := Fe.GetDict()
	fontInfo := d.FontInfo()

	switch fontInfo := fontInfo.(type) {
	case *dict.FontInfoGlyfEmbedded:
		body, err := pdf.GetStreamReader(r, fontInfo.Ref)
		if err != nil {
			return nil
		}
		info, err := sfnt.Read(body)
		if err != nil {
			return nil
		}
		outlines, ok := info.Outlines.(*glyf.Outlines)
		if !ok {
			return nil
		}

		m := make(map[cid.CID]string)

		// method 1: use glyph names, if present
		if outlines.Names != nil {
			if fontInfo.CIDToGID != nil {
				for cidVal, gid := range fontInfo.CIDToGID {
					if int(gid) > len(outlines.Names) {
						continue
					}
					name := outlines.Names[gid]
					if name == "" {
						continue
					}

					text := names.ToUnicode(name, fontInfo.PostScriptName)
					m[cid.CID(cidVal)] = text
				}
			}
		}
		return m
	default:
		return nil
	}
}

type affine struct {
	intercept, slope float64
}

var commonCharacters = map[string]affine{
	" ": {0, 1},
	" ": {0, 1},
	")": {-43.01937, 1.0268},
	"/": {-10.99708, 0.9623335},
	"•": {-24.2725, 0.9956384},
	"−": {-439.6255, 1.238626},
	"∗": {91.30598, 0.7265824},
	"1": {-130.7855, 0.9746186},
	"a": {-131.2164, 0.9740258},
	"A": {72.40703, 0.4928694},
	"e": {-136.5258, 0.9895894},
	"E": {-28.76257, 0.6957778},
	"i": {51.62929, 0.8973944},
	"ε": {-56.25771, 0.9947787},
	"Ω": {-132.9966, 1.002173},
	"中": {-356.8609, 1.215483},
}

func spaceWidthHeuristic(dict font.Dict) float64 {
	guesses := []float64{280}
	for _, info := range dict.Characters() {
		if coef, ok := commonCharacters[info.Text]; ok && info.Width > 0 {
			guesses = append(guesses, coef.intercept+coef.slope*info.Width)
		}
	}
	slices.Sort(guesses)

	// calculate the median
	var guess float64
	n := len(guesses)
	if n%2 == 0 {
		guess = (guesses[n/2-1] + guesses[n/2]) / 2
	} else {
		guess = guesses[n/2]
	}

	// adjustment to remove empirical bias
	guess = 1.366239*guess - 139.183703

	// clamp to approximate [0.01, 0.99] quantile range
	if guess < 200 {
		guess = 200
	} else if guess > 1000 {
		guess = 1000
	}

	return guess
}
