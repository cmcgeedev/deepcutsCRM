package domain

import "testing"

func TestOrderTransitions(t *testing.T) {
	allowed := map[OrderStatus][]OrderStatus{
		OrderDraft:     {OrderConfirmed, OrderCancelled},
		OrderConfirmed: {OrderScheduled, OrderCancelled, OrderDraft},
		OrderScheduled: {OrderConfirmed, OrderDelivered, OrderCancelled},
		OrderDelivered: {OrderFinalized},
		OrderFinalized: {},
		OrderCancelled: {},
	}
	all := []OrderStatus{OrderDraft, OrderConfirmed, OrderScheduled, OrderDelivered, OrderFinalized, OrderCancelled}
	for from, tos := range allowed {
		ok := map[OrderStatus]bool{}
		for _, to := range tos {
			ok[to] = true
		}
		for _, to := range all {
			if got := from.CanTransition(to); got != ok[to] {
				t.Errorf("%s→%s: got %v want %v", from, to, got, ok[to])
			}
		}
	}
}

func TestRouteTransitions(t *testing.T) {
	if !RoutePlanned.CanTransition(RouteOut) || !RouteOut.CanTransition(RouteComplete) {
		t.Error("forward transitions must be allowed")
	}
	if RoutePlanned.CanTransition(RouteComplete) || RouteOut.CanTransition(RoutePlanned) || RouteComplete.CanTransition(RouteOut) {
		t.Error("skips and reversals must be rejected")
	}
}

func TestLineEditScope(t *testing.T) {
	cases := []struct {
		o    OrderStatus
		r    RouteStatus
		want LineEditScope
	}{
		{OrderDraft, "", EditAll},
		{OrderConfirmed, "", EditAll},
		{OrderScheduled, RoutePlanned, EditAll},
		{OrderScheduled, RouteOut, EditShippedAndPrice},
		{OrderDelivered, RouteOut, EditDelivered},
		{OrderDelivered, RouteComplete, EditDelivered},
		{OrderFinalized, RouteComplete, EditNone},
		{OrderCancelled, "", EditNone},
	}
	for _, c := range cases {
		if got := LineEditScopeFor(c.o, c.r); got != c.want {
			t.Errorf("%s/%s: got %v want %v", c.o, c.r, got, c.want)
		}
	}
}
