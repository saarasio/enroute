// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package sync

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/saarasio/enroute/enroutectl/config"
	"testing"
)

type Assert struct {
	t *testing.T
}

func Equal(t *testing.T, want, got interface{}) {
	t.Helper()
	Assert{t}.Equal(want, got)
}

// Equal will call t.Fatal if want and got are not equal.
func (a Assert) Equal(want, got interface{}) {
	a.t.Helper()
	opts := []cmp.Option{
		cmpopts.AcyclicTransformer("UnmarshalAny", unmarshalAny),
		// errors to be equal only if both are nil or both are non-nil.
		cmp.Comparer(func(x, y error) bool {
			return (x == nil) == (y == nil)
		}),
	}
	diff := cmp.Diff(want, got, opts...)
	if diff != "" {
		a.t.Fatal(diff)
	}
}

func unmarshalAny(a *any.Any) proto.Message {
	if a == nil {
		return nil
	}
	pb, err := ptypes.Empty(a)
	if err != nil {
		panic(err.Error())
	}
	err = ptypes.UnmarshalAny(a, pb)
	if err != nil {
		panic(err.Error())
	}
	return pb
}

func TestUpstreamCreate(t *testing.T) {

	pua := PostUpstreamArg{
		Upstream_name:   "test",
		Upstream_ip:     "test.abc.com",
		Upstream_port:   "80",
		Upstream_weight: "100",
	}

	ps := config.ProxyServices{
		Service: config.Service{
			Fqdn:        "test",
			ServiceName: "test",
			Routes: []config.Routes{
				config.Routes{
					RouteName:   "rname",
					RoutePrefix: "/",
					RouteUpstreams: []config.RouteUpstreams{
						config.RouteUpstreams{
							config.Upstream{
								UpstreamName: "test",
								UpstreamIP:   "test.abc.com",
								UpstreamPort: 80,
							},
						},
					},
				},
			},
		},
	}
	rt := config.Routes{}

	tests := map[string]struct {
		in_ps  config.ProxyServices
		in_rt  config.Routes
		in_rt2 config.Routes
		want   PostUpstreamArg
	}{
		"Add Upstream": {
			in_ps:  ps,
			in_rt:  rt,
			in_rt2: rt,
			want:   pua,
		},
	}

	endrv := EnDrv{
		Dbg:     true,
		Offline: true,
	}

	esync := EnrouteSync{
		EnDrv: endrv,
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := esync.addServiceTree(tc.in_ps)
			Equal(t, tc.want, *got.pua)
		})
	}
}
