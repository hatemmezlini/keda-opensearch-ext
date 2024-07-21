package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	pb "github.com/hatemmezlini/keda-opensearch-ext/externalscaler"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	// "log"
	// "net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	esURL      = os.Getenv("ES_URL")
	esUsername = os.Getenv("ES_USERNAME")
	esPassword = os.Getenv("ES_PASSWORD")
)

// TemplateRequest struct to define the template request
type TemplateRequest struct {
	ID     string                 `json:"id"`
	Params map[string]interface{} `json:"params"`
}

// parseParams parses the input string and returns a map of parameters
func parseParams(params string) (map[string]interface{}, error) {
	paramMap := make(map[string]interface{})
	if params == "" {
		return paramMap, nil
	}
	pairs := strings.Split(params, ";")
	for _, pair := range pairs {
		kv := strings.Split(pair, ":")
		if len(kv) != 2 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parameter format: %v", pair)
		}
		paramMap[kv[0]] = kv[1]
	}
	return paramMap, nil
}

// getValueFromJSON extracts a value from the JSON response based on the given value location string
func getValueFromJSON(jsonData map[string]interface{}, valueLocation string) (int, error) {
	keys := strings.Split(valueLocation, ".")
	var rawValue interface{} = jsonData
	var ok bool
	for _, key := range keys {
		if rawValue, ok = rawValue.(map[string]interface{})[key]; !ok {
			return 0, status.Errorf(codes.NotFound, "value not found for key: %v", key)
		}
	}
	value, ok := rawValue.(float64)
	if !ok {
		return 0, status.Errorf(codes.Internal, "value is not an integer: %v", value)
	}
	return int(value), nil
}

func executeSearchTemplate(unsafeSSL bool, index, searchTemplateName, parameters, valueLocation string) (int, error) {
	paramMap, err := parseParams(parameters)
	if err != nil {
		return 0, err
	}

	templateRequest := TemplateRequest{
		ID:     searchTemplateName,
		Params: paramMap,
	}

	// Serialize the request payload to JSON
	requestBody, err := json.Marshal(templateRequest)
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}

	// Create a new HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/_search/template", esURL, index), bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}

	// Set the headers and basic auth
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(esUsername, esPassword)

	// Create a custom Transport that disables SSL verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: unsafeSSL}, // Disable SSL verification
	}
	// Create a custom HTTP client with the custom Transport
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second, // Set a timeout if needed
	}
	// Execute the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}

	// Check if the response status is OK
	if resp.StatusCode != http.StatusOK {
		return 0, status.Error(codes.Internal, err.Error())
	}

	// Parse the JSON response
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return 0, status.Errorf(codes.Internal, "error unmarshaling JSON response: %v", err)
	}

	// Extract the desired value from the JSON response
	value, err := getValueFromJSON(jsonResponse, valueLocation)
	if err != nil {
		return 0, err
	}
	return value, nil
}

type ExternalScaler struct{}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {
	index := scaledObject.ScalerMetadata["index"]
	searchTemplateName := scaledObject.ScalerMetadata["searchTemplateName"]
	parameters := scaledObject.ScalerMetadata["parameters"]
	// Parse unsafeSSL
	unsafeSSLstr, ok := scaledObject.ScalerMetadata["unsafeSSL"]
	if !ok {
		unsafeSSLstr = "false"
	}
	unsafeSSLlower := strings.ToLower(unsafeSSLstr)
	if unsafeSSLlower != "true" && unsafeSSLlower != "false" {
		return nil, status.Error(codes.InvalidArgument, "unsafeSSL must be either true or false")
	}
	unsafeSSL := unsafeSSLlower == "true"
	valueLocation := scaledObject.ScalerMetadata["valueLocation"]
	// Parse activationTargetValueStr
	activationTargetValueStr, ok := scaledObject.ScalerMetadata["activationTargetValue"]
	activationTargetValue := 0 // Default value
	if ok {
		var err error
		activationTargetValue, err = strconv.Atoi(activationTargetValueStr)
		if err != nil {
			// If conversion fails, default to 0
			activationTargetValue = 0
		}
	}

	if len(index) == 0 || len(searchTemplateName) == 0 || len(parameters) == 0 || len(valueLocation) == 0 {
		return nil, status.Error(codes.InvalidArgument, "index, searchTemplateName, parameters and valueLocation must be specified")
	}

	value, err := executeSearchTemplate(unsafeSSL, index, searchTemplateName, parameters, valueLocation)
	if err != nil {
		return nil, err
	}

	return &pb.IsActiveResponse{
		Result: value > activationTargetValue,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {
	targetValueStr := scaledObject.ScalerMetadata["targetValue"]
	var targetValue int64 = 50 // Default value
	if len(targetValueStr) > 0 {
		var err error
		targetValue, err = strconv.ParseInt(targetValueStr, 10, 64)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "targetValue must be an int64")
		}
	}
	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{
			{
				MetricName: "searchMatchDocCount",
				TargetSize: targetValue,
			},
		},
	}, nil
}

func (e *ExternalScaler) GetMetrics(ctx context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	index := metricRequest.ScaledObjectRef.ScalerMetadata["index"]
	searchTemplateName := metricRequest.ScaledObjectRef.ScalerMetadata["searchTemplateName"]
	parameters := metricRequest.ScaledObjectRef.ScalerMetadata["parameters"]
	// Parse unsafeSSL
	unsafeSSLstr, ok := metricRequest.ScaledObjectRef.ScalerMetadata["unsafeSSL"]
	if !ok {
		unsafeSSLstr = "false"
	}
	unsafeSSLlower := strings.ToLower(unsafeSSLstr)
	if unsafeSSLlower != "true" && unsafeSSLlower != "false" {
		return nil, status.Error(codes.InvalidArgument, "unsafeSSL must be either true or false")
	}
	unsafeSSL := unsafeSSLlower == "true"
	valueLocation := metricRequest.ScaledObjectRef.ScalerMetadata["valueLocation"]
	if len(index) == 0 || len(searchTemplateName) == 0 || len(parameters) == 0 || len(valueLocation) == 0 {
		return nil, status.Error(codes.InvalidArgument, "index, searchTemplateName, parameters and valueLocation must be specified")
	}
	value, err := executeSearchTemplate(unsafeSSL, index, searchTemplateName, parameters, valueLocation)
	if err != nil {
		return nil, err
	}
	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  "searchMatchDocCount",
			MetricValue: int64(value),
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {
	index := scaledObject.ScalerMetadata["index"]
	searchTemplateName := scaledObject.ScalerMetadata["searchTemplateName"]
	parameters := scaledObject.ScalerMetadata["parameters"]
	// Parse unsafeSSL
	unsafeSSLstr, ok := scaledObject.ScalerMetadata["unsafeSSL"]
	if !ok {
		unsafeSSLstr = "false"
	}
	unsafeSSLlower := strings.ToLower(unsafeSSLstr)
	if unsafeSSLlower != "true" && unsafeSSLlower != "false" {
		return status.Error(codes.InvalidArgument, "unsafeSSL must be either true or false")
	}
	unsafeSSL := unsafeSSLlower == "true"
	valueLocation := scaledObject.ScalerMetadata["valueLocation"]
	// Parse activationTargetValue
	activationTargetValueStr, ok := scaledObject.ScalerMetadata["activationTargetValue"]
	activationTargetValue := 0 // Default value
	if ok {
		var err error
		activationTargetValue, err = strconv.Atoi(activationTargetValueStr)
		if err != nil {
			// If conversion fails, default to 0
			activationTargetValue = 0
		}
	}
	if len(index) == 0 || len(searchTemplateName) == 0 || len(parameters) == 0 || len(valueLocation) == 0 {
		return status.Error(codes.InvalidArgument, "index, searchTemplateName, parameters and valueLocation must be specified")
	}
	for {
		select {
		case <-epsServer.Context().Done():
			// call cancelled
			return nil
		case <-time.Tick(time.Minute * 20):
			value, err := executeSearchTemplate(unsafeSSL, index, searchTemplateName, parameters, valueLocation)
			if err != nil {
				// log error
			} else if value > activationTargetValue {
				err = epsServer.Send(&pb.IsActiveResponse{
					Result: true,
				})
			}
		}
	}
}

// readinessHandler checks connection to Opensearch
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	// Try to get a basic response from Elasticsearch
	req, err := http.NewRequest("GET", esURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.SetBasicAuth(esUsername, esPassword)
	// Create a custom Transport that disables SSL verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Disable SSL verification
	}
	// Create a custom HTTP client with the custom Transport
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second, // Set a timeout if needed
	}
	// Execute the HTTP request
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Opensearch is not ready", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// livenessHandler always returns "OK"
func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":6000")
	pb.RegisterExternalScalerServer(grpcServer, &ExternalScaler{})

	// HTTP server for readiness and liveness endpoints
	http.HandleFunc("/readiness", readinessHandler)
	http.HandleFunc("/liveness", livenessHandler)

	go func() {
		fmt.Println("Starting HTTP server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	fmt.Println("Started GRPC server on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
