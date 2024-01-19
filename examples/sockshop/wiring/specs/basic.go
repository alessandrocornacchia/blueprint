// Package specs implements wiring specs for the SockShop application.
//
// The wiring spec can be specified using the -w option when running wiring/main.go
package specs

import (
	"github.com/blueprint-uservices/blueprint/blueprint/pkg/wiring"
	"github.com/blueprint-uservices/blueprint/plugins/cmdbuilder"
	"github.com/blueprint-uservices/blueprint/plugins/simple"
	"github.com/blueprint-uservices/blueprint/plugins/workflow"
)

// A simple wiring spec that compiles all services to a single process and therefore directly invoke each other.
// No RPC, containers, processes etc. are used.
var Basic = cmdbuilder.SpecOption{
	Name:        "basic",
	Description: "A basic single-process wiring spec with no modifiers",
	Build:       makeBasicSpec,
}

func makeBasicSpec(spec wiring.WiringSpec) ([]string, error) {
	user_db := simple.NoSQLDB(spec, "user_db")
	user_service := workflow.Service(spec, "user_service", "UserService", user_db)

	payment_service := workflow.Service(spec, "payment_service", "PaymentService", "500")

	cart_db := simple.NoSQLDB(spec, "cart_db")
	cart_service := workflow.Service(spec, "cart_service", "CartService", cart_db)

	shipqueue := simple.Queue(spec, "shipping_queue")
	shipdb := simple.NoSQLDB(spec, "shipping_db")
	shipping_service := workflow.Service(spec, "shipping_service", "ShippingService", shipqueue, shipdb)

	queue_master := workflow.Service(spec, "queue_master", "QueueMaster", shipqueue, shipping_service)

	order_db := simple.NoSQLDB(spec, "order_db")
	order_service := workflow.Service(spec, "order_service", "OrderService", user_service, cart_service, payment_service, shipping_service, order_db)

	catalogue_db := simple.RelationalDB(spec, "catalogue_db")
	catalogue_service := workflow.Service(spec, "catalogue_service", "CatalogueService", catalogue_db)

	frontend := workflow.Service(spec, "frontend", "Frontend", user_service, catalogue_service, cart_service, order_service)

	return []string{user_service, payment_service, cart_service, shipping_service, queue_master, order_service, catalogue_service, frontend}, nil
}
