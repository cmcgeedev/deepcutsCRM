package domain

type OrderStatus string

const (
	OrderDraft     OrderStatus = "draft"
	OrderConfirmed OrderStatus = "confirmed"
	OrderScheduled OrderStatus = "scheduled"
	OrderDelivered OrderStatus = "delivered"
	OrderFinalized OrderStatus = "finalized"
	OrderCancelled OrderStatus = "cancelled"
)

var orderTransitions = map[OrderStatus]map[OrderStatus]bool{
	OrderDraft:     {OrderConfirmed: true, OrderCancelled: true},
	OrderConfirmed: {OrderScheduled: true, OrderCancelled: true, OrderDraft: true},
	OrderScheduled: {OrderConfirmed: true, OrderDelivered: true, OrderCancelled: true},
	OrderDelivered: {OrderFinalized: true},
}

func (s OrderStatus) CanTransition(to OrderStatus) bool { return orderTransitions[s][to] }

type RouteStatus string

const (
	RoutePlanned  RouteStatus = "planned"
	RouteOut      RouteStatus = "out"
	RouteComplete RouteStatus = "complete"
)

func (s RouteStatus) CanTransition(to RouteStatus) bool {
	return (s == RoutePlanned && to == RouteOut) || (s == RouteOut && to == RouteComplete)
}

type StopStatus string

const (
	StopPending   StopStatus = "pending"
	StopDelivered StopStatus = "delivered"
	StopSkipped   StopStatus = "skipped"
)

// LineEditScope says which order-line fields the office may change.
type LineEditScope int

const (
	EditNone            LineEditScope = iota // finalized or cancelled
	EditShippedAndPrice                      // route is out: shipped weight and price override only
	EditAll                                  // draft, confirmed, or scheduled on a planned route
	EditDelivered                            // delivered, awaiting finalize: delivered qty/weight and notes
)

// LineEditScopeFor takes the order status and the status of its active route ("" when none).
func LineEditScopeFor(order OrderStatus, route RouteStatus) LineEditScope {
	switch order {
	case OrderDraft, OrderConfirmed:
		return EditAll
	case OrderScheduled:
		if route == RouteOut {
			return EditShippedAndPrice
		}
		return EditAll
	case OrderDelivered:
		return EditDelivered
	}
	return EditNone
}
