package saaras

import (
	"testing"

	ir "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
)

func TestConvertPathRouteMatchToDagRouteMatch(t *testing.T) {

	saarasr1 := SaarasRoute2{}
	irCond1 := ir.Condition{Prefix: "/"}

	saarasr2 := SaarasRoute2{Route_prefix: "/test2"}
	irCond2 := ir.Condition{Prefix: "/test2"}

	saarasr3 := SaarasRoute2{Route_config: `{"Prefix" : "/test3"}`}
	irCond3 := ir.Condition{Prefix: "/test3"}

	saarasr4 := SaarasRoute2{Route_config: `
    {
        "Prefix" : "/test4",
        "Header":
        [
          { "header_name": ":method", "header_value" : "GET" }
        ]
    }
    `}
	irCond4 := ir.Condition{Prefix: "/test4"}
	irCond5 := ir.Condition{Header: &ir.HeaderCondition{Name: ":method", Contains: "GET"}}

	tests := map[string]struct {
		route SaarasRoute2
		want  []ir.Condition
	}{
		"Empty route match condition": {
			route: saarasr1,
			want:  []ir.Condition{irCond1},
		},
		"Prefix only": {
			route: saarasr2,
			want:  []ir.Condition{irCond2},
		},
		"Prefix in route config": {
			route: saarasr3,
			want:  []ir.Condition{irCond3},
		},
		"Prefix and Method": {
			route: saarasr4,
			want:  []ir.Condition{irCond4, irCond5},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := saaras_routecondition_to_v1b1_ir_routecondition(tc.route)
			assert.Equal(t, tc.want, got)
		})
	}
}
