package geocode

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	earthRadiusM    = 6_371_000
	metersPerDegree = 111_320
	maxRadiusM      = 500_000 // 500 km
)

var radiusPattern = regexp.MustCompile(`(?i)^\s*(\d+(?:\.\d+)?)\s*(m|km|mi)\s*$`)

func parseRadius(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalidRadius
	}
	m := radiusPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, ErrInvalidRadius
	}
	value, err := strconv.ParseFloat(m[1], 64)
	if err != nil || value <= 0 {
		return 0, ErrInvalidRadius
	}
	var meters float64
	switch strings.ToLower(m[2]) {
	case "m":
		meters = value
	case "km":
		meters = value * 1000
	case "mi":
		meters = value * 1609.344
	default:
		return 0, ErrInvalidRadius
	}
	if meters > maxRadiusM {
		return 0, ErrInvalidRadius
	}
	return meters, nil
}

func boundingBox(lat, lon, radiusM float64) (minLat, maxLat, minLon, maxLon float64) {
	deltaLat := radiusM / metersPerDegree
	cosLat := math.Cos(lat * math.Pi / 180)
	if cosLat < 1e-9 {
		cosLat = 1e-9
	}
	deltaLon := radiusM / (metersPerDegree * cosLat)

	minLat = lat - deltaLat
	maxLat = lat + deltaLat
	minLon = lon - deltaLon
	maxLon = lon + deltaLon
	return minLat, maxLat, minLon, maxLon
}

func haversineM(lat1, lon1, lat2, lon2 float64) float64 {
	const deg2rad = math.Pi / 180
	dLat := (lat2 - lat1) * deg2rad
	dLon := (lon2 - lon1) * deg2rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*deg2rad)*math.Cos(lat2*deg2rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}

func roundDistanceM(m float64) float64 {
	return math.Round(m*10) / 10
}