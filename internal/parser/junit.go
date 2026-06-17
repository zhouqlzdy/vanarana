package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"vanarana/internal/model"
)

func ParseJunit(r io.Reader) (*model.JunitMetrics, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse junit html: %w", err)
	}

	m := &model.JunitMetrics{}

	doc.Find("#summary .infoBox").Each(func(_ int, s *goquery.Selection) {
		id, _ := s.Attr("id")
		counter := strings.TrimSpace(s.Find(".counter").Text())
		switch id {
		case "tests":
			m.TotalTests, _ = strconv.Atoi(counter)
		case "failures":
			m.Failures, _ = strconv.Atoi(counter)
		case "ignored":
			m.Ignored, _ = strconv.Atoi(counter)
		case "duration":
			m.DurationMs = parseDuration(counter)
		}
	})

	rateText := doc.Find("#successRate .percent").Text()
	m.SuccessRate = parsePercent(rateText)

	doc.Find("#tab0 table tbody tr").Each(func(_ int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() < 6 {
			return
		}
		pkg := model.PackageJunit{
			Name:     strings.TrimSpace(cells.Eq(0).Find("a").Text()),
			Tests:    atoi(cells.Eq(1).Text()),
			Failures: atoi(cells.Eq(2).Text()),
			Ignored:  atoi(cells.Eq(3).Text()),
		}
		pkg.DurationMs = parseDuration(cells.Eq(4).Text())
		if pkg.Name == "" {
			return
		}
		m.Packages = append(m.Packages, pkg)
	})

	return m, nil
}

func parseDuration(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	if strings.Contains(s, "m") {
		parts := strings.SplitN(s, "m", 2)
		minutes, _ := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		secsStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), "s")
		seconds, _ := strconv.ParseFloat(secsStr, 64)
		return minutes*60*1000 + int64(seconds*1000)
	}

	if strings.HasSuffix(s, "s") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64)
		return int64(val * 1000)
	}

	return 0
}

func parsePercent(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
