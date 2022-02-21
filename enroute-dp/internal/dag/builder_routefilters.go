package dag

import (
	// "fmt"

	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
)

func (b *builder) lookupRouteFilter(m RouteFilterMeta) *RouteFilter {
	rf, ok := b.routefilters[m]
	if !ok {
		return nil
	}
	return rf
}

func (b *builder) lookupHTTPRouteFilter(m RouteFilterMeta) *RouteFilter {
	rf := b.lookupRouteFilter(m)

	if rf == nil {
		rf_k8s, ok := b.source.routefilters[m]
		if !ok {
			return nil
		}
		rf = b.addRouteFilter(rf_k8s, m)
	}

	return rf
}

func (b *builder) addRouteFilter(rf_k8s *gatewayhostv1.RouteFilter, m RouteFilterMeta) *RouteFilter {
	if b.routefilters == nil {
		b.routefilters = make(map[RouteFilterMeta]*RouteFilter)
	}

	rf_dag := RouteFilter{
		Filter: Filter{
			Filter_name:   rf_k8s.Spec.Name,
			Filter_type:   rf_k8s.Spec.Type,
			Filter_config: rf_k8s.Spec.RouteFilterConfig.Config,
		},
	}

	b.routefilters[m] = &rf_dag

	return &rf_dag
}

func (b *builder) RouteServiceFilters(dag_r *Route, k8s_r *gatewayhostv1.Route, ns string) []*RouteFilter {

	var rfilters []*RouteFilter

	if k8s_r != nil && k8s_r.Filters != nil {
		if len(k8s_r.Filters) > 0 {
			for _, f := range k8s_r.Filters {
				if f.Type == saarasconfig.FILTER_TYPE_RT_CIRCUITBREAKERS {
					m := RouteFilterMeta{filter_type: f.Type, name: f.Name, namespace: ns}
					rf := b.lookupHTTPRouteFilter(m)
					if rf != nil && dag_r != nil {
						if dag_r.RouteFilters == nil {
							dag_r.RouteFilters = make([]*RouteFilter, 0)
						}
						rfilters = append(rfilters, rf)
					}
				}
			}
		}
	}

	return rfilters
}

func (b *builder) SetupRouteFilters(dag_r *Route, k8s_r *gatewayhostv1.Route, ns string) {

	if k8s_r != nil && k8s_r.Filters != nil {
		if len(k8s_r.Filters) > 0 {
			for _, f := range k8s_r.Filters {
				m := RouteFilterMeta{filter_type: f.Type, name: f.Name, namespace: ns}
				rf := b.lookupHTTPRouteFilter(m)
				if rf != nil && dag_r != nil {
					if dag_r.RouteFilters == nil {
						dag_r.RouteFilters = make([]*RouteFilter, 0)
					}
					dag_r.RouteFilters = append(dag_r.RouteFilters, rf)
				}
			}
		}
	}
}
