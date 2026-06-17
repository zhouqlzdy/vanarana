package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"vanarana/internal/model"
)

func ParseJacoco(r io.Reader) (*model.JacocoMetrics, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse jacoco html: %w", err)
	}

	m := &model.JacocoMetrics{}

	tfoot := doc.Find("tfoot tr td")
	if tfoot.Length() < 13 {
		return nil, fmt.Errorf("unexpected jacoco tfoot structure: got %d cells", tfoot.Length())
	}

	m.InstructionCoverage = parsePercent(tfoot.Eq(2).Text())
	m.BranchCoverage = parsePercent(tfoot.Eq(4).Text())

	linesMissed := cleanNum(tfoot.Eq(7).Text())
	linesTotal := cleanNum(tfoot.Eq(8).Text())
	m.LinesMissed = linesMissed
	m.LinesTotal = linesTotal
	if linesTotal > 0 {
		m.LineCoverage = float64(linesTotal-linesMissed) / float64(linesTotal) * 100
	}

	methodsMissed := cleanNum(tfoot.Eq(9).Text())
	methodsTotalv := cleanNum(tfoot.Eq(10).Text())
	if methodsTotalv > 0 {
		m.MethodCoverage = float64(methodsTotalv-methodsMissed) / float64(methodsTotalv) * 100
	}

	doc.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() < 9 {
			return
		}

		pkgName := strings.TrimSpace(cells.Find("a.el_package").Text())
		if pkgName == "" {
			return
		}

		instCov := parsePercent(strings.TrimSpace(cells.Eq(2).Text()))
		branchCov := parsePercent(strings.TrimSpace(cells.Eq(4).Text()))
		lm := cleanNum(cells.Eq(7).Text())
		lt := cleanNum(cells.Eq(8).Text())

		var lineCov float64
		if lt > 0 {
			lineCov = float64(lt-lm) / float64(lt) * 100
		}

		m.Packages = append(m.Packages, model.PackageCoverage{
			Name:                pkgName,
			InstructionCoverage: instCov,
			BranchCoverage:      branchCov,
			LineCoverage:        lineCov,
			LinesTotal:          lt,
			LinesMissed:         lm,
		})
	})

	return m, nil
}

func cleanNum(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.Atoi(s)
	return n
}
