package utils

import (
	"math"
)

const (
	EarthRadiusKm = 6371.0
	EarthRadiusM  = 6371000.0
	DegToRad      = math.Pi / 180.0
	RadToDeg      = 180.0 / math.Pi
)

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type BoundingBox struct {
	NorthEast Coordinate `json:"northEast"`
	SouthWest Coordinate `json:"southWest"`
}

type GeofenceCircle struct {
	Center Coordinate `json:"center"`
	Radius float64    `json:"radius"` // in meters
}

// CalculateDistance calculates the distance between two coordinates using the Haversine formula
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * DegToRad
	lon1Rad := lon1 * DegToRad
	lat2Rad := lat2 * DegToRad
	lon2Rad := lon2 * DegToRad

	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusM * c
}

// CalculateBearing calculates the bearing between two coordinates
func CalculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * DegToRad
	lon1Rad := lon1 * DegToRad
	lat2Rad := lat2 * DegToRad
	lon2Rad := lon2 * DegToRad

	dlon := lon2Rad - lon1Rad

	y := math.Sin(dlon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dlon)

	bearing := math.Atan2(y, x) * RadToDeg
	return math.Mod(bearing+360, 360)
}

// IsWithinGeofence checks if a coordinate is within a circular geofence
func IsWithinGeofence(lat, lon float64, geofence GeofenceCircle) bool {
	distance := CalculateDistance(lat, lon, geofence.Center.Latitude, geofence.Center.Longitude)
	return distance <= geofence.Radius
}

// CalculateBoundingBox calculates a bounding box around a center point with a given radius
func CalculateBoundingBox(centerLat, centerLon, radiusM float64) BoundingBox {
	// Convert radius from meters to degrees (approximately)
	latDelta := radiusM / 111000.0 // 1 degree latitude ≈ 111km
	lonDelta := radiusM / (111000.0 * math.Cos(centerLat*DegToRad))

	return BoundingBox{
		NorthEast: Coordinate{
			Latitude:  centerLat + latDelta,
			Longitude: centerLon + lonDelta,
		},
		SouthWest: Coordinate{
			Latitude:  centerLat - latDelta,
			Longitude: centerLon - lonDelta,
		},
	}
}

// IsValidCoordinate checks if latitude and longitude values are valid
func IsValidCoordinate(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

// NormalizeCoordinate ensures coordinates are within valid ranges
func NormalizeCoordinate(lat, lon float64) (float64, float64) {
	// Normalize latitude
	if lat > 90 {
		lat = 90
	} else if lat < -90 {
		lat = -90
	}

	// Normalize longitude
	lon = math.Mod(lon+180, 360) - 180
	if lon < -180 {
		lon = -180
	} else if lon > 180 {
		lon = 180
	}

	return lat, lon
}

// CalculateGeofenceEvents determines if a location change represents an entry or exit
func CalculateGeofenceEvents(oldLat, oldLon, newLat, newLon float64, geofences []GeofenceCircle) []GeofenceEvent {
	var events []GeofenceEvent

	for i, geofence := range geofences {
		wasInside := IsWithinGeofence(oldLat, oldLon, geofence)
		isInside := IsWithinGeofence(newLat, newLon, geofence)

		if !wasInside && isInside {
			events = append(events, GeofenceEvent{
				GeofenceIndex: i,
				EventType:     "enter",
				Distance:      CalculateDistance(newLat, newLon, geofence.Center.Latitude, geofence.Center.Longitude),
			})
		} else if wasInside && !isInside {
			events = append(events, GeofenceEvent{
				GeofenceIndex: i,
				EventType:     "exit",
				Distance:      CalculateDistance(newLat, newLon, geofence.Center.Latitude, geofence.Center.Longitude),
			})
		}
	}

	return events
}

type GeofenceEvent struct {
	GeofenceIndex int     `json:"geofenceIndex"`
	EventType     string  `json:"eventType"` // "enter" or "exit"
	Distance      float64 `json:"distance"`
}

// CalculateSpeed calculates speed between two points given the time difference
func CalculateSpeed(lat1, lon1 float64, time1 int64, lat2, lon2 float64, time2 int64) float64 {
	distance := CalculateDistance(lat1, lon1, lat2, lon2)
	timeDiff := float64(time2 - time1) // in seconds

	if timeDiff <= 0 {
		return 0
	}

	return distance / timeDiff // meters per second
}

// ConvertSpeedUnits converts speed between different units
func ConvertSpeedUnits(speed float64, fromUnit, toUnit string) float64 {
	// Convert to m/s first
	var mps float64
	switch fromUnit {
	case "mps":
		mps = speed
	case "kmh":
		mps = speed / 3.6
	case "mph":
		mps = speed * 0.44704
	default:
		return speed // Return original if unknown unit
	}

	// Convert from m/s to target unit
	switch toUnit {
	case "mps":
		return mps
	case "kmh":
		return mps * 3.6
	case "mph":
		return mps / 0.44704
	default:
		return mps // Return m/s if unknown unit
	}
}

// DetectDriving attempts to detect if the user is driving based on speed and other factors
func DetectDriving(speed float64, accuracy float64) bool {
	// Consider driving if speed > 5 km/h (1.39 m/s) and accuracy is reasonable
	speedThreshold := 1.39    // m/s (approximately 5 km/h)
	accuracyThreshold := 50.0 // meters

	return speed > speedThreshold && accuracy <= accuracyThreshold
}

// CalculateCenter calculates the center point of multiple coordinates
func CalculateCenter(coordinates []Coordinate) Coordinate {
	if len(coordinates) == 0 {
		return Coordinate{0, 0}
	}

	var latSum, lonSum float64
	for _, coord := range coordinates {
		latSum += coord.Latitude
		lonSum += coord.Longitude
	}

	return Coordinate{
		Latitude:  latSum / float64(len(coordinates)),
		Longitude: lonSum / float64(len(coordinates)),
	}
}

// CalculateArea calculates the approximate area of a polygon in square meters
func CalculateArea(coordinates []Coordinate) float64 {
	if len(coordinates) < 3 {
		return 0
	}

	// Use the shoelace formula for polygon area
	var area float64
	n := len(coordinates)

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += coordinates[i].Latitude * coordinates[j].Longitude
		area -= coordinates[j].Latitude * coordinates[i].Longitude
	}

	area = math.Abs(area) / 2.0

	// Convert from degrees to square meters (approximate)
	// 1 degree ≈ 111,000 meters
	return area * 111000 * 111000
}

// GetQuadrant returns the quadrant (NE, NW, SE, SW) of coord2 relative to coord1
func GetQuadrant(coord1, coord2 Coordinate) string {
	if coord2.Latitude >= coord1.Latitude && coord2.Longitude >= coord1.Longitude {
		return "NE"
	} else if coord2.Latitude >= coord1.Latitude && coord2.Longitude < coord1.Longitude {
		return "NW"
	} else if coord2.Latitude < coord1.Latitude && coord2.Longitude >= coord1.Longitude {
		return "SE"
	} else {
		return "SW"
	}
}

// IsPointInPolygon checks if a point is inside a polygon using ray casting algorithm
func IsPointInPolygon(point Coordinate, polygon []Coordinate) bool {
	if len(polygon) < 3 {
		return false
	}

	x, y := point.Longitude, point.Latitude
	inside := false

	j := len(polygon) - 1
	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i].Longitude, polygon[i].Latitude
		xj, yj := polygon[j].Longitude, polygon[j].Latitude

		if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}

	return inside
}

// CalculateHeading calculates heading from bearing (0-360 degrees)
func CalculateHeading(bearing float64) string {
	// Normalize bearing to 0-360
	bearing = math.Mod(bearing+360, 360)

	directions := []string{
		"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW",
	}

	index := int(math.Round(bearing/22.5)) % 16
	return directions[index]
}
