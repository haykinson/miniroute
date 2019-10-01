package main

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/kellydunn/golang-geo"
	"github.com/stretchr/testify/assert"
	"github.com/weilunwu/go-geofence"
)

func TestCorrectness(t *testing.T) {
	polygon := randomPolygon(2000, 0.1)
	geoPoly := geo.NewPolygon(polygon)
	holes := []*geo.Point{}
	geofence := geofence.NewGeofence([][]*geo.Point{polygon, holes}, int64(20))

	for i := 0; i < 100000; i++ {
		point := randomPoint(200)
		assert.Equal(t, geofence.Inside(point), geoPoly.Contains(point))
	}
}

func TestHoles(t *testing.T) {
	polygon := []*geo.Point{
		geo.NewPoint(0, 0),
		geo.NewPoint(4, 0),
		geo.NewPoint(4, 4),
		geo.NewPoint(0, 4),
	}

	hole := []*geo.Point{
		geo.NewPoint(1, 0),
		geo.NewPoint(3, 0),
		geo.NewPoint(3, 3),
		geo.NewPoint(1, 3),
	}

	geofence := geofence.NewGeofence([][]*geo.Point{polygon, hole})
	assert.True(t, geofence.Inside(geo.NewPoint(0.5, 0.5)))
	assert.False(t, geofence.Inside(geo.NewPoint(2, 2)))
	assert.False(t, geofence.Inside(geo.NewPoint(5, 5)))
}


func BenchmarkGeoContains(b *testing.B) {
	// Chicago geofence
	polygon := []*geo.Point{
		geo.NewPoint(42.01313565896657, -87.89133314508945),
		geo.NewPoint(42.01086525470408, -87.94498134870082),
		geo.NewPoint(41.955566495567936, -87.94566393946297),
		geo.NewPoint(41.937218295745865, -87.88581848144531),
		geo.NewPoint(41.96295962052549, -87.86811594385654),
		geo.NewPoint(41.93385557339662, -87.86333084106445),
		geo.NewPoint(41.934494079111666, -87.81011581420898),
		geo.NewPoint(41.90554916282452, -87.80925750732422),
		geo.NewPoint(41.9058827519221, -87.77938842773438),
		geo.NewPoint(41.86402837073972, -87.77792931126896),
		geo.NewPoint(41.864284053216565, -87.75638580846135),
		geo.NewPoint(41.82348977579423, -87.75552751729265),
		geo.NewPoint(41.823042045417644, -87.80410768697038),
		geo.NewPoint(41.771468158020106, -87.80324938008562),
		geo.NewPoint(41.772364335324305, -87.74625778198242),
		geo.NewPoint(41.730894639311565, -87.74513235432096),
		geo.NewPoint(41.73166805909664, -87.6870346069336),
		geo.NewPoint(41.71748939617332, -87.68600471266836),
		geo.NewPoint(41.716966221614854, -87.7243280201219),
		geo.NewPoint(41.69405798811367, -87.72351264953613),
		geo.NewPoint(41.693865716655395, -87.74385454365984),
		geo.NewPoint(41.67463566843159, -87.74299623677507),
		geo.NewPoint(41.67550471265456, -87.6654052734375),
		geo.NewPoint(41.651683859743336, -87.66489028930664),
		geo.NewPoint(41.65181212480582, -87.64789581298828),
		geo.NewPoint(41.652036588050684, -87.62532234191895),
		geo.NewPoint(41.643100214173714, -87.62506484985352),
		geo.NewPoint(41.643492184875946, -87.51889228820801),
		geo.NewPoint(41.642929165686375, -87.38588330335915),
		geo.NewPoint(41.836600482955916, -87.43858338799328),
		geo.NewPoint(42.05042567111704, -87.40253437310457),
		geo.NewPoint(42.070116505457364, -87.47205723077059),
		geo.NewPoint(42.0681413002819, -87.66792302951217),
		geo.NewPoint(42.02862488227374, -87.66551960259676),
		geo.NewPoint(42.0280511074349, -87.71289814263582),
		geo.NewPoint(41.998468275360544, -87.71301263943315),
		geo.NewPoint(41.9988509912138, -87.75069231167436),
		geo.NewPoint(42.02100207763309, -87.77704238542356),
		geo.NewPoint(42.02010937741473, -87.831029893714),
		geo.NewPoint(41.98719839843826, -87.83120155116194),
		geo.NewPoint(41.9948536336077, -87.86373138340423),
	}

	golangGeo := geo.NewPolygon(polygon)
	for i := 0; i < b.N; i++ {
		point := randomPointCustom(41.642929165686375, 42.070116505457364, -87.94566393946297, -87.38588330335915, 100)
		golangGeo.Contains(point)
	}
}

func randomPoint(length float64) *geo.Point {
	return geo.NewPoint(rand.Float64()*length-length/2, rand.Float64()*length-length/2)
}

func randomPolygon(length float64, percentageOfLength float64) []*geo.Point {
	polygon := make([]*geo.Point, 1000)
	for i := 0; i < 1000; i++ {
		polygon[i] = randomPoint(length * percentageOfLength)
	}
	return polygon
}

func randomPointCustom(minLat float64, maxLat float64, minLng float64, maxLng float64, factor float64) *geo.Point {
	latRange := maxLat - minLat
	lngRange := maxLng - minLng
	return geo.NewPoint((minLat+maxLat)/2-latRange*factor/2+latRange*factor*rand.Float64(), (minLng+maxLng)/2-lngRange*factor/2+lngRange*factor*rand.Float64())
}

func testGeofence() (int, int) {
	// Chicago geofence
	polygon := []*geo.Point{
		geo.NewPoint(42.01313565896657, -87.89133314508945),
		geo.NewPoint(42.01086525470408, -87.94498134870082),
		geo.NewPoint(41.955566495567936, -87.94566393946297),
		geo.NewPoint(41.937218295745865, -87.88581848144531),
		geo.NewPoint(41.96295962052549, -87.86811594385654),
		geo.NewPoint(41.93385557339662, -87.86333084106445),
		geo.NewPoint(41.934494079111666, -87.81011581420898),
		geo.NewPoint(41.90554916282452, -87.80925750732422),
		geo.NewPoint(41.9058827519221, -87.77938842773438),
		geo.NewPoint(41.86402837073972, -87.77792931126896),
		geo.NewPoint(41.864284053216565, -87.75638580846135),
		geo.NewPoint(41.82348977579423, -87.75552751729265),
		geo.NewPoint(41.823042045417644, -87.80410768697038),
		geo.NewPoint(41.771468158020106, -87.80324938008562),
		geo.NewPoint(41.772364335324305, -87.74625778198242),
		geo.NewPoint(41.730894639311565, -87.74513235432096),
		geo.NewPoint(41.73166805909664, -87.6870346069336),
		geo.NewPoint(41.71748939617332, -87.68600471266836),
		geo.NewPoint(41.716966221614854, -87.7243280201219),
		geo.NewPoint(41.69405798811367, -87.72351264953613),
		geo.NewPoint(41.693865716655395, -87.74385454365984),
		geo.NewPoint(41.67463566843159, -87.74299623677507),
		geo.NewPoint(41.67550471265456, -87.6654052734375),
		geo.NewPoint(41.651683859743336, -87.66489028930664),
		geo.NewPoint(41.65181212480582, -87.64789581298828),
		geo.NewPoint(41.652036588050684, -87.62532234191895),
		geo.NewPoint(41.643100214173714, -87.62506484985352),
		geo.NewPoint(41.643492184875946, -87.51889228820801),
		geo.NewPoint(41.642929165686375, -87.38588330335915),
		geo.NewPoint(41.836600482955916, -87.43858338799328),
		geo.NewPoint(42.05042567111704, -87.40253437310457),
		geo.NewPoint(42.070116505457364, -87.47205723077059),
		geo.NewPoint(42.0681413002819, -87.66792302951217),
		geo.NewPoint(42.02862488227374, -87.66551960259676),
		geo.NewPoint(42.0280511074349, -87.71289814263582),
		geo.NewPoint(41.998468275360544, -87.71301263943315),
		geo.NewPoint(41.9988509912138, -87.75069231167436),
		geo.NewPoint(42.02100207763309, -87.77704238542356),
		geo.NewPoint(42.02010937741473, -87.831029893714),
		geo.NewPoint(41.98719839843826, -87.83120155116194),
		geo.NewPoint(41.9948536336077, -87.86373138340423),
	}

	fence := geofence.NewGeofence([][]*geo.Point{polygon, []*geo.Point{}})
	b := 10000000
	ok := 0
	notok := 0
	for i := 0; i < b; i++ {
		point := randomPointCustom(41.9, 42.02, -87.8, -87.8, 100)
		result := fence.Inside(point)
		if result {
			ok++
		} else {
			notok++
		}
	}

	return ok, notok
}

func main() {
	fmt.Println("Hello, world.")
	numok, numnotok := testGeofence()
	fmt.Println("tested, ok/notok: ", numok, numnotok)
}