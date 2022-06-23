//go:build !e && !c
// +build !e,!c

package dag

import (
	_ "github.com/davecgh/go-spew/spew"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/internal/logger"
	"github.com/sirupsen/logrus"
)

func (b *builder) lookupHTTPFilter(m HttpFilterMeta) *HttpFilter {
	hf, ok := b.httpfilters[m]
	if !ok {
		return nil
	}
	return hf
}

func (b *builder) addHttpFilter(hf_k8s *gatewayhostv1.HttpFilter, m HttpFilterMeta, ir *gatewayhostv1.GatewayHost) *HttpFilter {
	if b.httpfilters == nil {
		b.httpfilters = make(map[HttpFilterMeta]*HttpFilter)
	}

	if logger.EL.ELogger != nil && logger.EL.ELogger.GetLevel() >= logrus.InfoLevel {
		logger.EL.ELogger.Debugf("internal:dag:builder_httpfilter:addHttpFilter() GH [%s:%s] HTTPFilter [%s] [%s]\n", ir.Namespace, ir.Name, hf_k8s.Spec.Name, hf_k8s.Spec.Type)
	}

	hf_dag := HttpFilter{
		Filter: Filter{
			Filter_name:   hf_k8s.Spec.Name,
			Filter_type:   hf_k8s.Spec.Type,
			Filter_config: hf_k8s.Spec.HttpFilterConfig.Config,
		},
	}

	b.httpfilters[m] = &hf_dag

	return &hf_dag
}

func (b *builder) lookupHTTPVHFilter(m HttpFilterMeta, ir *gatewayhostv1.GatewayHost) *HttpFilter {
	hf := b.lookupHTTPFilter(m)

	if hf == nil {
		hf_k8s, ok := b.source.httpfilters[m]
		if !ok {
			return nil
		}
		hf = b.addHttpFilter(hf_k8s, m, ir)
	}

	return hf
}

func (b *builder) SetupHttpFilters(dag_vh *VirtualHost, ir *gatewayhostv1.GatewayHost) {

	k8s_vh := ir.Spec.VirtualHost
	ns := ir.Namespace

	if k8s_vh != nil && k8s_vh.Filters != nil {
		if len(k8s_vh.Filters) > 0 {
			for _, f := range k8s_vh.Filters {
				m := HttpFilterMeta{filter_type: f.Type, name: f.Name, namespace: ns}
				hf := b.lookupHTTPVHFilter(m, ir)
				if logger.EL.ELogger != nil && logger.EL.ELogger.GetLevel() >= logrus.DebugLevel && dag_vh != nil && hf != nil {
					logger.EL.ELogger.Debugf("internal:dag:builder_httpfilter:SetupHttpFilters() GH [%s:%s] VH [%s] HttpFilters [%s] [%s]\n", ir.Namespace, ir.Name, dag_vh.Name, hf.Filter.Filter_name, hf.Filter.Filter_type)
				}
				if hf != nil && dag_vh != nil {
					if dag_vh.HttpFilters == nil {
						dag_vh.HttpFilters = make([]*HttpFilter, 0)
					}
					dag_vh.HttpFilters = append(dag_vh.HttpFilters, hf)
				}
			}
		}
	}
}
