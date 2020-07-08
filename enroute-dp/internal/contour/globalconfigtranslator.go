// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package contour

import (
	"github.com/sirupsen/logrus"
	// v1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ingressroutev1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	_ "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	k8scache "k8s.io/client-go/tools/cache"
)

type GlobalConfigTranslator struct {
	logrus.FieldLogger
	clusterLoadAssignmentCache
	Cond
	RateLimitSyncChannel chan string
}

func (e *GlobalConfigTranslator) OnAdd(obj interface{}) {
	switch obj := obj.(type) {
	case *ingressroutev1.GlobalConfig:
		e.addGlobalConfig(obj)
	default:
		e.Errorf("OnAdd unexpected type %T: %#v", obj, obj)
	}
}

func (e *GlobalConfigTranslator) OnUpdate(oldObj, newObj interface{}) {
	switch newObj := newObj.(type) {
	case *ingressroutev1.GlobalConfig:
		oldObj, ok := oldObj.(*ingressroutev1.GlobalConfig)
		if !ok {
			e.Errorf("OnUpdate GlobalConfig %#v received invalid oldObj %T; %#v", newObj, oldObj, oldObj)
			return
		}
		e.updateGlobalConfig(oldObj, newObj)
	default:
		e.Errorf("OnUpdate unexpected type %T: %#v", newObj, newObj)
	}
}

func (e *GlobalConfigTranslator) OnDelete(obj interface{}) {
	switch obj := obj.(type) {
	case *ingressroutev1.GlobalConfig:
		e.removeGlobalConfig(obj)
	case k8scache.DeletedFinalStateUnknown:
		e.OnDelete(obj.Obj) // recurse into ourselves with the tombstoned value
	default:
		e.Errorf("OnDelete unexpected type %T: %#v", obj, obj)
	}
}

func (e *GlobalConfigTranslator) addGlobalConfig(pc *ingressroutev1.GlobalConfig) {
	switch pc.Spec.Type {
	default:
	}
}

func (e *GlobalConfigTranslator) updateGlobalConfig(oldpc, newpc *ingressroutev1.GlobalConfig) {
	switch newpc.Spec.Type {
	default:
	}
}

func (e *GlobalConfigTranslator) removeGlobalConfig(pc *ingressroutev1.GlobalConfig) {

	switch pc.Spec.Type {
	default:
	}
}
