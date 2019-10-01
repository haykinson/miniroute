package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	geo "github.com/kellydunn/golang-geo"
	"github.com/shopspring/decimal"
	"github.com/weilunwu/go-geofence"
	"io/ioutil"
	"log"
	"os"
)


/**
{"Id":3756239,"Rcvr":11156,"HasSig":true,"Sig":22,"Icao":"3950CF","Bad":false,"Reg":"F-GUGP",
"FSeen":"\/Date(1559378428473)\/","TSecs":3,"CMsgs":15,"Alt":18400,"GAlt":18657,
"InHg":30.177166,"AltT":0,"Call":"AFR112H","Lat":45.557696,"Long":11.823212,"PosTime":1559378429830,
"Mlat":false,"Tisb":false,"Spd":441.8,"Trak":276.5,"TrkH":false,"Type":"A318","Mdl":"Airbus A318 111",
"Man":"Airbus","CNum":"2967","From":"EDDM Munich, Germany","To":"LFPG Charles de Gaulle, Paris, France",
"Op":"Air France","OpIcao":"AFR","Sqk":"0240","Help":false,"Vsi":2304,"VsiT":0,"WTC":2,"Species":1,
"Engines":"2","EngType":3,"EngMount":0,"Mil":false,"Cou":"France","HasPic":false,"Interested":false,
"FlightsCount":0,"Gnd":false,"SpdTyp":0,"CallSus":false,"ResetTrail":true,"TT":"a","Trt":2,"Year":"2006",
"Cos":[45.557602,11.824639,1559378429454.0,18500.0,45.557696,11.823212,1559378429830.0,18500.0]}
 */

type Entries struct {
	AcList []Entry `json:acList`
}

type Entry struct {
	Id int
	Reg string
	Alt int
	GAlt int
	Trak decimal.Decimal
	Lat decimal.Decimal
	Long decimal.Decimal
}


func parseZip(zipFile *zip.File) (*Entries, error) {
	jsonFile, err := zipFile.Open()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer jsonFile.Close()

	bytes,_ := ioutil.ReadAll(jsonFile)

	var entries Entries
	json.Unmarshal(bytes, &entries)

	return &entries, nil
}


func defineNorth() *geofence.Geofence {
	polygon := []*geo.Point{
		geo.NewPoint(33.9519407,	-118.4362650),
		geo.NewPoint(33.9590602,	-118.3768702),
		geo.NewPoint(33.9752902,	-118.3785868),
		geo.NewPoint(33.9630468,	-118.4462214),
	}

	fence := geofence.NewGeofence([][]*geo.Point{polygon, []*geo.Point{}})
	return fence
}

func defineSouth() *geofence.Geofence {
	polygon := []*geo.Point{
		geo.NewPoint(33.9285849,	-118.4249353),
		geo.NewPoint(33.9345669,	-118.3772135),
		geo.NewPoint(33.9234571,	-118.3744669),
		geo.NewPoint(33.9154800,	-118.4184122),
	}

	fence := geofence.NewGeofence([][]*geo.Point{polygon, []*geo.Point{}})
	return fence
}



func evaluate(filename string, entries *Entries, north *geofence.Geofence, south *geofence.Geofence) {
	for _, entry := range entries.AcList {
		if len(entry.Reg) == 0 || entry.Lat.IsZero() || entry.Long.IsZero() {
			continue
		}

		if (entry.Alt < 2250 && entry.GAlt < 2250) || (entry.Alt > 2750 && entry.GAlt > 2750) {
			continue
		}

		lat, _ := entry.Lat.Float64()
		long, _ := entry.Long.Float64()
		point := geo.NewPoint(lat, long)

		if north.Inside(point) {
			fmt.Println(filename, "North", entry.Reg, entry.Lat, entry.Long, entry.Alt, entry.GAlt, entry.Trak)
		}
		if south.Inside(point) {
			fmt.Println(filename, "South", entry.Reg, entry.Lat, entry.Long, entry.Alt, entry.GAlt, entry.Trak)
		}

	}
}


func processZipFile(zipFile *zip.File, north *geofence.Geofence, south *geofence.Geofence, sem chan bool) {
	entries, err := parseZip(zipFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	evaluate(zipFile.Name, entries, north, south)

	<- sem
}

func processZip(zipFilename string, north *geofence.Geofence, south *geofence.Geofence) {
	// Open a zip archive for reading.
	r, err := zip.OpenReader(zipFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	concurrency := 8
	sem := make(chan bool, concurrency)

	for _, f := range r.File {
		fmt.Println(f.Name)

		sem <- true
		go processZipFile(f, north, south, sem)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

}

func main() {
	north := defineNorth()
	south := defineSouth()

	processZip("/Users/ilya/Downloads/adsb/2019-06-01.zip", north, south)

	input := bufio.NewScanner(os.Stdin)
	input.Scan()

}
