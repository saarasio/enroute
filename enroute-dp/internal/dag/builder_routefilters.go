package dag

import (
	// "fmt"

	ingressroutev1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	//"github.com/davecgh/go-spew/spew"
)

//func v1b1RouteFilterToDagRouteFilter(ir_rl *ingressroutev1.RouteFilter) *RouteFilter {
//	if ir_rl == nil {
//		return nil
//	}
//	return &RouteFilter{Filters: ir_rl.Filters}
//}

func (b *builder) lookupRouteFilter(m RouteFilterMeta) *cfg.SaarasRouteFilter {
	// fmt.Printf("lookupRouteFilter() %+v \n", m)
	rf, ok := b.routefilters[m]
	if !ok {
		return nil
	}
	return rf
}

func (b *builder) lookupHTTPRouteFilter(m RouteFilterMeta) *cfg.SaarasRouteFilter {
	rf := b.lookupRouteFilter(m)

	if rf == nil {
		rf_k8s, ok := b.source.routefilters[m]
		// fmt.Printf("lookupHTTPRouteFilter() Lookup of +%v returned +%v\n", m, rf_k8s)
		// fmt.Printf("lookupHTTPRouteFilter() Cache contents +%v\n", b.source.routefilters)
		if !ok {
			// fmt.Printf("lookupHTTPRouteFilter() lookup in k8s cache failed\n")
			return nil
		}
		rf = b.addRouteFilter(rf_k8s, m)
	}

	return rf
}

func (b *builder) addRouteFilter(rf_k8s *ingressroutev1.RouteFilter, m RouteFilterMeta) *cfg.SaarasRouteFilter {
	if b.routefilters == nil {
		b.routefilters = make(map[RouteFilterMeta]*cfg.SaarasRouteFilter)
	}

	rf_dag := &cfg.SaarasRouteFilter{
		Filter_name:   rf_k8s.Spec.Name,
		Filter_type:   rf_k8s.Spec.Type,
		Filter_config: rf_k8s.Spec.RouteFilterConfig.Config,
	}
	b.routefilters[m] = rf_dag
	// fmt.Printf("addRouteFilter() Add route filter to route on rf_dag [%+v] rf_k8s [%+v]\n", rf_dag, rf_k8s)

	return rf_dag
}

func (b *builder) SetupRouteFilters(dag_r *Route, k8s_r *ingressroutev1.Route, ns string) {

	// fmt.Printf("SetupRouteFilters() k8s route - %+v\n", k8s_r)
	if k8s_r != nil && k8s_r.Filters != nil {
		if len(k8s_r.Filters) > 0 {
			for _, f := range k8s_r.Filters {
				m := RouteFilterMeta{filter_type: f.Type, name: f.Name, namespace: ns}
				// fmt.Printf("SetupRouteFilters() Looking up %+v \n", m)
				rf := b.lookupHTTPRouteFilter(m)
				// fmt.Printf("SetupRouteFilters() Lookup of %+v returned +%v\n", m, rf)
				if rf != nil && dag_r != nil {
					if dag_r.RouteFilters == nil {
						dag_r.RouteFilters = &RouteFilter{}
					}
					if dag_r.RouteFilters.Filters == nil {
						dag_r.RouteFilters.Filters = make([]*cfg.SaarasRouteFilter, 0)
					}
					dag_r.RouteFilters.Filters = append(dag_r.RouteFilters.Filters, rf)
				}
			}
		}
	}

}
