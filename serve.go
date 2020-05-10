package main

import (
	context "context"
	"flag"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"html/template"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	keyMiniroute = "miniroute"
	keySFRA = "sfra"
	keyUnknown = "k"
)

type flightServiceServer struct {
	UnimplementedFlightServiceServer

	redisClient   *redis.Client
	expiration    time.Duration
	lastHeartbeat time.Time
	messageCount  uint64
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}

	return false
}

func (f *flightServiceServer) Heartbeat(ctx context.Context, in *HeartbeatRequest) (*HeartbeatResponse, error) {
	ts, err := ptypes.Timestamp(in.Timestamp)
	if err == nil {
		f.lastHeartbeat = ts
	}
	f.messageCount += in.MessageCount
	return &HeartbeatResponse{}, nil
}

func (f *flightServiceServer) RecordDetectedFlight(ctx context.Context, in *RecordDetectedFlightRequest) (*RecordDetectedFlightResponse, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !(ok && headers["auth"] != nil && contains(headers["auth"], "password")) {
		return nil, status.Error(codes.PermissionDenied, "Permission denied")
	}

	bytes, _ := proto.Marshal(in)

	var key string
	if in.Corridor == RecordDetectedFlightRequest_CORRIDOR_MINIROUTE {
		key = keyMiniroute
	} else if in.Corridor == RecordDetectedFlightRequest_CORRIDOR_SFRA {
		key = keySFRA
	} else {
		key = keyUnknown
	}

	f.redisClient.LPush(key, bytes)
	f.redisClient.Expire(key, f.expiration)
	f.redisClient.LTrim(key, 0, int64(numToList))

	logger.Println(in)

	return &RecordDetectedFlightResponse{}, nil
}

func initGrpc(flightService *flightServiceServer) {
	lis, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%v", grpcListen))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	RegisterFlightServiceServer(grpcServer, flightService)
	reflection.Register(grpcServer)
	grpcServer.Serve(lis)
}

type DataForTemplate struct {
	Title         string
	MessageCount  uint64
	LastHeartbeat time.Time
	FlightsMiniroute       []*RecordDetectedFlightRequest
	FlightsSFRA       []*RecordDetectedFlightRequest
}

func (f *flightServiceServer) handleWebRequest(w http.ResponseWriter, r *http.Request) {
	resultsMiniroute := f.redisClient.LRange(keyMiniroute, 0, int64(numToList))
	resultsSFRA := f.redisClient.LRange(keySFRA, 0, int64(numToList))

	flightsMiniroute, err := unmarshalFlightsFromRedis(resultsMiniroute)
	if err != nil {
		fmt.Fprint(w, "Error")
		logger.Printf("Error while unmarshalling miniroute: %v\n", err)
		return
	}

	flightsSFRA, err := unmarshalFlightsFromRedis(resultsSFRA)
	if err != nil {
		fmt.Fprint(w, "Error")
		logger.Printf("Error while unmarshalling sfra: %v\n", err)
		return
	}

	logger.Printf("flights: miniroute: %v, sfra: %v\n", flightsMiniroute, flightsSFRA)

	t, _ := template.
		New("template.html").
		Funcs(template.FuncMap{"timestampConverter": ptypes.TimestampString}).
		ParseFiles("template.html")

	err = t.Execute(w, DataForTemplate{Title: "foo", FlightsMiniroute: flightsMiniroute, FlightsSFRA: flightsSFRA, LastHeartbeat: f.lastHeartbeat, MessageCount: f.messageCount})
	if err != nil {
		logger.Println(err)
	}
}

func unmarshalFlightsFromRedis(results *redis.StringSliceCmd) ([]*RecordDetectedFlightRequest, error) {
	flights := make([]*RecordDetectedFlightRequest, 0)
	for _, result := range results.Val() {
		flight := &RecordDetectedFlightRequest{}
		if err := proto.Unmarshal([]byte(result), flight); err != nil {
			logger.Fatal("Could not unmarshal stored proto")
			return nil, err
		}

		flights = append(flights, flight)
	}
	return flights, nil
}

func newService() flightServiceServer {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return flightServiceServer{
		redisClient: client,
		expiration:  expireFlightRecordsIn,
	}
}

var (
	grpcListen            int
	webListen             int
	numToList             int
	expireFlightRecordsIn time.Duration
)

func initServingFlags() {
	flag.IntVar(&webListen, "web", 8080, "Listen http on a given port")
	flag.IntVar(&grpcListen, "grpc", 50051, "Listen grpc on a given port")
	flag.IntVar(&numToList, "num", 20, "Number of recent items to include in the listing")

	defaultDuration, _ := time.ParseDuration("5m")
	flag.DurationVar(&expireFlightRecordsIn, "expire", defaultDuration, "Time to consider flights current")
	flag.Parse()
}

func main() {
	initServingFlags()

	initLogging()

	flightService := newService()

	go initGrpc(&flightService)

	http.HandleFunc("/", flightService.handleWebRequest)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", webListen), nil))
}
