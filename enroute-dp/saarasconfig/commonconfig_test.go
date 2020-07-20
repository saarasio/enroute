package saarasconfig

import (
	"testing"

	"github.com/saarasio/enroute/enroute-dp/internal/assert"
)

// docurl: https://getenroute.io/blog/why-every-api-needs-a-clock/
// docurl: https://getenroute.io/docs/getting-started-enroute-ingress-controller/

func TestRateLimitRouteFilterUnmarshal(t *testing.T) {
	tests := map[string]struct {
	    filter_config   string
		want            RouteActionDescriptors
    }{

        "one descriptor": {
            filter_config :  `
            {
                "descriptors":
                [
                { "remote_address" : "{}" }
                ]
            }
            `,
            want: RouteActionDescriptors{
                Descriptors : []Descriptors{{
                    RemoteAddress : "{}",
                },},
            },
        },
        "two descriptors": {
            filter_config :  `
            {
                "descriptors":
                [
                { "remote_address" : "{}" },
                { "source_cluster" : "{}" }
                ]
            }
            `,
            want: RouteActionDescriptors{
                Descriptors : []Descriptors{
                    { RemoteAddress : "{}" },
                    { SourceCluster: "{}" },
                },
            },
        },
        "three descriptors": {
            filter_config :  `
            {
                "descriptors":
                [
                { "remote_address" : "{}" },
                { "source_cluster" : "{}" },
                { "destination_cluster" : "{}" }
                ]
            }
            `,
            want: RouteActionDescriptors{
                Descriptors : []Descriptors{
                    { RemoteAddress : "{}" },
                    { SourceCluster: "{}", },
                    { DestinationCluster: "{}" },
                },
            },
        },
        "four descriptors": {
            filter_config :  `
            {
                "descriptors":
                [
                { "remote_address" : "{}" },
                { "source_cluster" : "{}" },
                { "destination_cluster" : "{}" },
			    { "generic_key" : { "descriptor_value" : "blah" } }
                ]
            }
            `,
			want: RouteActionDescriptors{
				Descriptors : []Descriptors{
					{RemoteAddress : "{}"},
					{SourceCluster: "{}"},
					{DestinationCluster: "{}"},
					{GenericKey: &GenericKeyType{DescriptorValue: "blah"}},
				},
			},
        },
        "five descriptors": {
            filter_config :  `
            {
                "descriptors":
                [
                { "remote_address" : "{}" },
                { "source_cluster" : "{}" },
                { "destination_cluster" : "{}" },
			    { "generic_key" : { "descriptor_value" : "blah" } },
                { "request_headers" : { "header_name" : "blah", "descriptor_key" : "blah"} }
                ]
            }
            `,
			want: RouteActionDescriptors{
				Descriptors : []Descriptors{
					{RemoteAddress : "{}"},
					{SourceCluster: "{}"},
					{DestinationCluster: "{}"},
					{GenericKey: &GenericKeyType{DescriptorValue: "blah"}},
					{
						RequestHeaders: &RequestHeadersType{HeaderName: "blah", DescriptorKey: "blah"},
					},
				},
			},
        },
    }

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got,_ := UnmarshalRateLimitRouteFilterConfig(tc.filter_config)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRouteMatchUnmarshal(t *testing.T) {
    tests := map[string]struct {
        route_config    string
        want            RouteMatchConditions
    }{

        "GET with prefix /": {
            route_config :  `
            {   
                "Prefix" : "/",  
                "match":
                [
                { "header_name" : ":method", "header_value" : "GET"}
                ]
            }
            `,
            want: RouteMatchConditions{
                    Prefix: "/",
                MatchConditions: []RouteMatchCondition{{
                    HeaderName: ":method",
                    HeaderValue: "GET",
                },},
            },
        },
    }
    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            got,_ := UnmarshalRouteMatchCondition(tc.route_config)
            assert.Equal(t, tc.want, got)
        })
    }
}
