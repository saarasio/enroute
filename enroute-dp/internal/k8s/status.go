// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package k8s contains helpers for setting the GatewayHost status
package k8s

import (
	"encoding/json"
	"context"

	jsonpatch "github.com/evanphx/json-patch"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	clientset "github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/types"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatewayHostStatus allows for updating the object's Status field
type GatewayHostStatus struct {
	Client clientset.Interface
}

// SetStatus sets the GatewayHost status field to an Valid or Invalid status
func (irs *GatewayHostStatus) SetStatus(status, desc string, existing *gatewayhostv1.GatewayHost) error {
	// Check if update needed by comparing status & desc
	if existing.CurrentStatus != status || existing.Description != desc {
		updated := existing.DeepCopy()
		updated.Status = gatewayhostv1.Status{
			CurrentStatus: status,
			Description:   desc,
		}
		return irs.setStatus(existing, updated)
	}
	return nil
}

func (irs *GatewayHostStatus) setStatus(existing, updated *gatewayhostv1.GatewayHost) error {
	existingBytes, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	// Need to set the resource version of the updated endpoints to the resource
	// version of the current service. Otherwise, the resulting patch does not
	// have a resource version, and the server complains.
	updated.ResourceVersion = existing.ResourceVersion
	updatedBytes, err := json.Marshal(updated)
	if err != nil {
		return err
	}
	patchBytes, err := jsonpatch.CreateMergePatch(existingBytes, updatedBytes)
	if err != nil {
		return err
	}

	var po meta_v1.PatchOptions

	if irs != nil && irs.Client != nil && irs.Client.EnrouteV1beta1() != nil && existing != nil {
		_, err = irs.Client.EnrouteV1beta1().GatewayHosts(existing.GetNamespace()).Patch(context.TODO(), existing.GetName(), types.MergePatchType, patchBytes, po)
	}
	return err
}
