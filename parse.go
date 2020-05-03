package main

import (
	"fmt"
	geo "github.com/kellydunn/golang-geo"
	"github.com/ornen/go-sbs1"
	"github.com/weilunwu/go-geofence"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func defineNorth() *geofence.Geofence {
	polygon := []*geo.Point{
		geo.NewPoint(33.9519407, -118.4362650),
		geo.NewPoint(33.9590602, -118.3768702),
		geo.NewPoint(33.9752902, -118.3785868),
		geo.NewPoint(33.9630468, -118.4462214),
	}

	fence := geofence.NewGeofence([][]*geo.Point{polygon, []*geo.Point{}})
	return fence
}

func defineSouth() *geofence.Geofence {
	polygon := []*geo.Point{
		geo.NewPoint(33.9285849, -118.4249353),
		geo.NewPoint(33.9345669, -118.3772135),
		geo.NewPoint(33.9234571, -118.3744669),
		geo.NewPoint(33.9154800, -118.4184122),
	}

	fence := geofence.NewGeofence([][]*geo.Point{polygon, []*geo.Point{}})
	return fence
}

const (
	locationUnknown = iota
	locationNorth   = iota
	locationSouth   = iota
)

type measure struct {
	messagesReceived             int
	messagesWithErrors           int
	messagesWithAltitude         int
	messagesWithTrack            int
	messagesWithCallsign         int
	messagesWithLocation         int
	messagesWithMatchingAltitude int
	messagesWithMatchingLocation int
	flightsCreated               int
	flightsExpired               int
	flightsOutput                int
}

type flightInfo struct {
	Callsign   string
	Direction  int
	AltitudeOk bool
	Expires    time.Time
	InNorth    bool
	InSouth    bool

	FullCaptures int
	Processed    bool
}

type capturedRegistration struct {
	HexId    string
	Callsign string
}

type capturedTrack struct {
	HexId string
	Track float64
}

type capturedAltitude struct {
	HexId    string
	Altitude int32
}

type capturedLocation struct {
	HexId    string
	Location int
}

func (c capturedLocation) Key() string {
	return c.HexId
}

func (c capturedAltitude) Key() string {
	return c.HexId
}

func (c capturedTrack) Key() string {
	return c.HexId
}

func (c capturedRegistration) Key() string {
	return c.HexId
}

type keyedCapture interface {
	Key() string
}

func outputFlight(output chan *flightInfo) {
	for flight := range output {
		var directionType string
		switch flight.Direction {
		case locationSouth:
			directionType = "South"
		case locationNorth:
			directionType = "North"
		default:
			directionType = "Unknown"
		}
		fmt.Printf("flightInfo: Reg %s, Moving: %s\n", flight.Callsign, directionType)
	}
}

func processFlights(flights map[string]*flightInfo, mutex *sync.Mutex, expiry time.Duration, captures chan keyedCapture, measures chan measure) {
	output := make(chan *flightInfo)
	go outputFlight(output)

	for capture := range captures {

		mutex.Lock()

		flight, ok := flights[capture.Key()]

		// if flightInfo was already processed, skip
		if ok && flight.Processed {
			continue
		}

		// consider this a new session if the flightInfo is very old
		if ok && flight.Expires.Before(time.Now()) {
			ok = false
		}

		// create a new record
		if !ok {
			flight = &flightInfo{
				Direction: locationUnknown,
				Expires:   time.Now().Add(expiry),
			}
			flights[capture.Key()] = flight
			measures <- measure{flightsCreated: 1}
		}

		switch e := capture.(type) {
		case capturedRegistration:
			flight.Callsign = e.Callsign
		case capturedTrack:
			flight.Direction = getDirection(e.Track)
		case capturedAltitude:
			flight.AltitudeOk = true
		case capturedLocation:
			if e.Location == locationNorth {
				flight.InNorth = true
			} else if e.Location == locationSouth {
				flight.InSouth = true
			}
		default:
			panic("unknown capture type")
		}

		if fullySatisfied(flight) {
			flight.FullCaptures++

			if flight.FullCaptures >= 2 {
				flight.Processed = true
				measures <- measure{flightsOutput: 1}
				output <- flight
			}
		}

		mutex.Unlock()
	}
}

func fullySatisfied(flight *flightInfo) bool {
	return len(flight.Callsign) > 0 &&
		flight.AltitudeOk &&
		flight.InNorth &&
		flight.InSouth &&
		flight.Direction != locationUnknown
}

func getDirection(track float64) int {
	if track >= 110 && track <= 160 {
		return locationSouth
	}
	if track >= 290 && track <= 350 {
		return locationNorth
	}

	return locationUnknown
}

func evaluateSBS1Message(message *sbs1.Message, north *geofence.Geofence, south *geofence.Geofence, captures chan keyedCapture, measures chan measure) bool {
	if message.MessageType != sbs1.MessageTypeTransmission {
		return false
	}

	recorded := false

	if transmissionTypesWithCallsign[message.TransmissionType] {
		captures <- capturedRegistration{
			HexId:    message.HexId,
			Callsign: message.Callsign,
		}
		measures <- measure{messagesWithCallsign: 1}
		recorded = true
	}

	if transmissionTypesWithAltitude[message.TransmissionType] {
		measures <- measure{messagesWithAltitude: 1}

		if message.Altitude >= 3250 && message.Altitude <= 4750 {
			captures <- capturedAltitude{
				HexId:    message.HexId,
				Altitude: message.Altitude,
			}
			measures <- measure{messagesWithMatchingAltitude: 1}
			recorded = true
		}
	}

	if transmissionTypesWithTrack[message.TransmissionType] {
		captures <- capturedTrack{
			HexId: message.HexId,
			Track: message.Track.Degrees(),
		}
		measures <- measure{messagesWithTrack: 1}
		recorded = true
	}

	if transmissionTypesWithCoords[message.TransmissionType] &&
		!(message.Coordinates.Lat.E5() == 0 && message.Coordinates.Lng.E5() == 0) {

		lat := message.Coordinates.Lat.Degrees()
		long := message.Coordinates.Lng.Degrees()
		point := geo.NewPoint(lat, long)
		location := locationUnknown

		if north.Inside(point) {
			location = locationNorth
			measures <- measure{messagesWithMatchingLocation: 1}
		} else if south.Inside(point) {
			location = locationSouth
			measures <- measure{messagesWithMatchingLocation: 1}
		}

		captures <- capturedLocation{
			HexId:    message.HexId,
			Location: location,
		}
		measures <- measure{messagesWithLocation: 1}
		recorded = true
	}

	return recorded
}

func processSBS1(north *geofence.Geofence, south *geofence.Geofence) {
	conn, err := net.Dial("tcp", "192.168.7.153:30003")

	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	var reader = sbs1.NewReader(conn)

	captures := make(chan keyedCapture)
	flights := make(map[string]*flightInfo)
	measures := make(chan measure)
	expiration, _ := time.ParseDuration("20m")
	cleanup, _ := time.ParseDuration("60m")

	cleanupTicker := time.NewTicker(1 * time.Minute)
	cleanupDone := make(chan bool)
	reportTicker := time.NewTicker(30 * time.Second)
	reportDone := make(chan bool)
	flightsMutex := &sync.Mutex{}

	go func() {
		for {
			select {
			case <-cleanupDone:
				return
			case <-cleanupTicker.C:
				flightsMutex.Lock()
				log.Println("Cleaning up flights...")
				removed := 0
				for key, flight := range flights {
					if time.Now().Sub(flight.Expires) > cleanup {
						delete(flights, key)
						removed++
					}
				}
				log.Printf("Removed %v flights\n", removed)
				flightsMutex.Unlock()

				measures <- measure{flightsExpired: removed}
			}
		}
	}()

	go func() {
		aggregateMeasure := measure{}
		for {
			select {
			case measure := <-measures:
				aggregateMeasure.messagesReceived += measure.messagesReceived
				aggregateMeasure.messagesWithErrors += measure.messagesWithErrors
				aggregateMeasure.flightsCreated += measure.flightsCreated
				aggregateMeasure.flightsExpired += measure.flightsExpired
				aggregateMeasure.flightsOutput += measure.flightsOutput
				aggregateMeasure.messagesWithAltitude += measure.messagesWithAltitude
				aggregateMeasure.messagesWithCallsign += measure.messagesWithCallsign
				aggregateMeasure.messagesWithLocation += measure.messagesWithLocation
				aggregateMeasure.messagesWithTrack += measure.messagesWithTrack
				aggregateMeasure.messagesWithMatchingAltitude += measure.messagesWithMatchingAltitude
				aggregateMeasure.messagesWithMatchingLocation += measure.messagesWithMatchingLocation
			case <-reportDone:
				return
			case <-reportTicker.C:
				flightsMutex.Lock()
				flightCount := 0
				satisfiedFlightCount := 0
				for _, flight := range flights {
					flightCount++
					if fullySatisfied(flight) {
						satisfiedFlightCount++
					} else {
						log.Println("Partial", flight)
					}
				}
				log.Printf("Flights: %v, satisfied: %v\n", flightCount, satisfiedFlightCount)
				flightsMutex.Unlock()

				log.Printf("%+v\n", aggregateMeasure)
			}
		}
	}()

	defer cleanupTicker.Stop()
	defer reportTicker.Stop()

	defer func() {
		cleanupDone <- true
		reportDone <- true
	}()

	go processFlights(flights, flightsMutex, expiration, captures, measures)

	for {
		var message, err = reader.Read()
		measures <- measure{messagesReceived: 1}

		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Println(err)
				measures <- measure{messagesWithErrors: 1}
				continue
			}
		}

		//log.Println(message)
		evaluateSBS1Message(message, north, south, captures, measures)
		//log.Printf("%v\n", recorded)
	}
}

var (
	transmissionTypesWithCallsign map[sbs1.TransmissionType]bool
	transmissionTypesWithAltitude map[sbs1.TransmissionType]bool
	transmissionTypesWithTrack    map[sbs1.TransmissionType]bool
	transmissionTypesWithCoords   map[sbs1.TransmissionType]bool
)

func initTransmissionTypes() {
	transmissionTypesWithCallsign = make(map[sbs1.TransmissionType]bool)
	transmissionTypesWithCallsign[sbs1.TransmissionTypeESIdentAndCategory] = true

	transmissionTypesWithAltitude = make(map[sbs1.TransmissionType]bool)
	transmissionTypesWithAltitude[sbs1.TransmissionTypeESSurfacePos] = true
	transmissionTypesWithAltitude[sbs1.TransmissionTypeESAirbornePos] = true
	transmissionTypesWithAltitude[sbs1.TransmissionTypeSurveillanceAlt] = true
	transmissionTypesWithAltitude[sbs1.TransmissionTypeSurveillanceId] = true
	transmissionTypesWithAltitude[sbs1.TransmissionTypeAirToAir] = true

	transmissionTypesWithTrack = make(map[sbs1.TransmissionType]bool)
	transmissionTypesWithTrack[sbs1.TransmissionTypeESSurfacePos] = true
	transmissionTypesWithTrack[sbs1.TransmissionTypeESAirborneVel] = true

	transmissionTypesWithCoords = make(map[sbs1.TransmissionType]bool)
	transmissionTypesWithCoords[sbs1.TransmissionTypeESSurfacePos] = true
	transmissionTypesWithCoords[sbs1.TransmissionTypeESAirbornePos] = true
}

func main() {
	north := defineNorth()
	south := defineSouth()
	initTransmissionTypes()

	processSBS1(north, south)
}
