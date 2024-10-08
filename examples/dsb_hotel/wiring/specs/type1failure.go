package specs

import (
	"github.com/blueprint-uservices/blueprint/blueprint/pkg/wiring"
	"github.com/blueprint-uservices/blueprint/examples/dsb_hotel/workflow/hotelreservation"
	"github.com/blueprint-uservices/blueprint/examples/dsb_hotel/workload/workloadgen"
	"github.com/blueprint-uservices/blueprint/plugins/cmdbuilder"
	"github.com/blueprint-uservices/blueprint/plugins/gotests"
	"github.com/blueprint-uservices/blueprint/plugins/jaeger"
	"github.com/blueprint-uservices/blueprint/plugins/memcached"
	"github.com/blueprint-uservices/blueprint/plugins/mongodb"
	"github.com/blueprint-uservices/blueprint/plugins/retries"
	"github.com/blueprint-uservices/blueprint/plugins/workflow"
	"github.com/blueprint-uservices/blueprint/plugins/workload"
)

// Wiring spec that represents the original configuration of the HotelReservation application.
// Each service is deployed in a separate container with all inter-service communication happening via GRPC.
// FrontEnd service provides a http frontend for making requests.
// All services are instrumented with opentelemetry tracing with spans being exported to a central Jaeger collector.
var Type1Failure = cmdbuilder.SpecOption{
	Name:        "type1failure",
	Description: "Deploys the DeathStarBench application with Type 1 metastable failure.",
	Build:       makeType1FailureSpec,
}

func makeType1FailureSpec(spec wiring.WiringSpec) ([]string, error) {
	var cntrs []string

	var allServices []string
	// Define backends
	trace_collector := jaeger.Collector(spec, "jaeger")
	user_db := mongodb.Container(spec, "user_db")
	recommendations_db := mongodb.Container(spec, "recomd_db")
	reserv_db := mongodb.Container(spec, "reserv_db")
	geo_db := mongodb.Container(spec, "geo_db")
	rate_db := mongodb.Container(spec, "rate_db")
	profile_db := mongodb.Container(spec, "profile_db")

	reserv_cache := memcached.Container(spec, "reserv_cache")
	rate_cache := memcached.Container(spec, "rate_cache")
	profile_cache := memcached.Container(spec, "profile_cache")

	// Define internal services
	user_service := workflow.Service[hotelreservation.UserService](spec, "user_service", user_db)
	retries.AddRetriesWithTimeouts(spec, "user_service", 10, "500ms") // Adds retries and timeouts BEFORE gRPC / HTTP
	user_ctr := applyDefaults(spec, user_service, trace_collector)
	cntrs = append(cntrs, user_ctr)
	allServices = append(allServices, "user_service")

	recomd_service := workflow.Service[hotelreservation.RecommendationService](spec, "recomd_service", recommendations_db)
	retries.AddRetriesWithTimeouts(spec, "recomd_service", 10, "500ms") // Adds retries and timeouts BEFORE gRPC / HTTP
	recomd_ctr := applyDefaults(spec, recomd_service, trace_collector)
	cntrs = append(cntrs, recomd_ctr)
	allServices = append(allServices, "recomd_service")

	reserv_service := workflow.Service[hotelreservation.ReservationService](spec, "reserv_service", reserv_cache, reserv_db)
	retries.AddRetriesWithTimeouts(spec, "reserv_service", 10, "500ms") // Adds retries and timeouts BEFORE gRPC / HTTP
	reserv_ctr := applyDefaults(spec, reserv_service, trace_collector)
	cntrs = append(cntrs, reserv_ctr)
	allServices = append(allServices, "reserv_service")

	geo_service := workflow.Service[hotelreservation.GeoService](spec, "geo_service", geo_db)
	retries.AddRetriesWithTimeouts(spec, "geo_service", 10, "500ms") // Adds retries and timeouts BEFORE gRPC / HTTP
	geo_ctr := applyDefaults(spec, geo_service, trace_collector)
	cntrs = append(cntrs, geo_ctr)
	allServices = append(allServices, "geo_service")

	rate_service := workflow.Service[hotelreservation.RateService](spec, "rate_service", rate_cache, rate_db)
	retries.AddRetriesWithTimeouts(spec, "rate_service", 10, "500ms") // Adds retries and timeouts BEFORE gRPC / HTTP
	rate_ctr := applyDefaults(spec, rate_service, trace_collector)
	cntrs = append(cntrs, rate_ctr)
	allServices = append(allServices, "rate_service")

	profile_service := workflow.Service[hotelreservation.ProfileService](spec, "profile_service", profile_cache, profile_db)
	retries.AddRetriesWithTimeouts(spec, "profile_service", 10, "500ms") // Adds retries and timeouts BEFORE gRPC / HTTP
	profile_ctr := applyDefaults(spec, profile_service, trace_collector)
	cntrs = append(cntrs, profile_ctr)
	allServices = append(allServices, "profile_service")

	search_service := workflow.Service[hotelreservation.SearchService](spec, "search_service", geo_service, rate_service)
	retries.AddRetriesWithTimeouts(spec, "search_service", 10, "500ms") // Adds retries and timeouts BEFORE gRPC / HTTP
	//latency.AddFixed(spec, "search_service", "500ms")
	search_ctr := applyDefaults(spec, search_service, trace_collector)
	cntrs = append(cntrs, search_ctr)
	allServices = append(allServices, "search_service")

	// Define frontend service
	frontend_service := workflow.Service[hotelreservation.FrontEndService](spec, "frontend_service", search_service, profile_service, recomd_service, user_service, reserv_service)
	frontend_ctr := applyHTTPDefaults(spec, frontend_service, trace_collector)
	cntrs = append(cntrs, frontend_ctr)
	allServices = append(allServices, "frontend_service")

	wlgen := workload.Generator[workloadgen.SimpleWorkload](spec, "wlgen", frontend_service)
	cntrs = append(cntrs, wlgen)

	tests := gotests.Test(spec, allServices...)
	cntrs = append(cntrs, tests)

	return cntrs, nil
}
