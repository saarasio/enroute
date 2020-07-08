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

func (b *builder) lookupHTTPFilter(m HttpFilterMeta) *cfg.SaarasRouteFilter {
	// fmt.Printf("lookupRouteFilter() %+v \n", m)
	hf, ok := b.httpfilters[m]
	if !ok {
		return nil
	}
	return hf
}

func (b *builder) lookupHTTPVHFilter(m HttpFilterMeta) *cfg.SaarasRouteFilter {
	hf := b.lookupHTTPFilter(m)

	if hf == nil {
		hf_k8s, ok := b.source.httpfilters[m]
		// fmt.Printf("lookupHTTPRouteFilter() Lookup of +%v returned +%v\n", m, rf_k8s)
		// fmt.Printf("lookupHTTPRouteFilter() Cache contents +%v\n", b.source.routefilters)
		if !ok {
			// fmt.Printf("lookupHTTPRouteFilter() lookup in k8s cache failed\n")
			return nil
		}
		hf = b.addHttpFilter(hf_k8s, m)
	}

	return hf
}

func (b *builder) addHttpFilter(hf_k8s *ingressroutev1.HttpFilter, m HttpFilterMeta) *cfg.SaarasRouteFilter {
	if b.httpfilters == nil {
		b.httpfilters = make(map[HttpFilterMeta]*cfg.SaarasRouteFilter)
	}

	hf_dag := &cfg.SaarasRouteFilter{
		Filter_name:   hf_k8s.Spec.Name,
		Filter_type:   hf_k8s.Spec.Type,
		Filter_config: hf_k8s.Spec.HttpFilterConfig.Config,
	}
	b.httpfilters[m] = hf_dag
	// fmt.Printf("addRouteFilter() Add route filter to route on rf_dag [%+v] rf_k8s [%+v]\n", rf_dag, rf_k8s)

	return hf_dag
}

func (b *builder) SetupHttpFilters(dag_vh *VirtualHost, k8s_vh *ingressroutev1.VirtualHost, ns string) {

	// fmt.Printf("SetupRouteFilters() k8s route - %+v\n", k8s_r)
	if k8s_vh != nil && k8s_vh.Filters != nil {
		if len(k8s_vh.Filters) > 0 {
			for _, f := range k8s_vh.Filters {
				m := HttpFilterMeta{filter_type: f.Type, name: f.Name, namespace: ns}
				// fmt.Printf("SetupRouteFilters() Looking up %+v \n", m)
				hf := b.lookupHTTPVHFilter(m)
				// fmt.Printf("SetupRouteFilters() Lookup of %+v returned +%v\n", m, rf)
				if hf != nil && dag_vh != nil {
					if dag_vh.HttpFilters == nil {
						dag_vh.HttpFilters = &HttpFilter{}
					}
					if dag_vh.HttpFilters.Filters == nil {
						dag_vh.HttpFilters.Filters = make([]*cfg.SaarasRouteFilter, 0)
					}
					dag_vh.HttpFilters.Filters = append(dag_vh.HttpFilters.Filters, hf)
				}
			}
		}
	}

}
