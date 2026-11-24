// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/apiserverendpoints"
)

func TestCacheByObjectWatchLabelSelector(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	// The ClusterObservability CRD is not installed.
	mapper := meta.NewDefaultRESTMapper(nil)
	for _, object := range []client.Object{
		&v1beta1.OpenTelemetryCollector{},
		&v1alpha1.TargetAllocator{},
		&v1alpha1.OpAMPBridge{},
		&v1alpha1.Instrumentation{},
	} {
		gvk, err := apiutil.GVKForObject(object, scheme)
		require.NoError(t, err)
		mapper.Add(gvk, meta.RESTScopeNamespace)
	}

	byObject, err := cacheByObject("", scheme, mapper)
	require.NoError(t, err)
	require.Len(t, byObject, 1)
	endpointSlice, endpointSliceSettings := apiserverendpoints.CacheByObject()
	for object, settings := range byObject {
		require.Equal(t, reflect.TypeOf(endpointSlice), reflect.TypeOf(object))
		require.Equal(t, endpointSliceSettings, settings)
	}

	byObject, err = cacheByObject("operator.example.com/instance=blue", scheme, mapper)
	require.NoError(t, err)
	require.Len(t, byObject, 5)

	wantTypes := map[reflect.Type]bool{
		reflect.TypeFor[*v1beta1.OpenTelemetryCollector](): true,
		reflect.TypeFor[*v1alpha1.TargetAllocator]():       true,
		reflect.TypeFor[*v1alpha1.OpAMPBridge]():           true,
		reflect.TypeFor[*v1alpha1.Instrumentation]():       true,
	}
	for object, settings := range byObject {
		if wantTypes[reflect.TypeOf(object)] {
			require.True(t, settings.Label.Matches(labels.Set{"operator.example.com/instance": "blue"}))
			require.False(t, settings.Label.Matches(labels.Set{"operator.example.com/instance": "green"}))
			delete(wantTypes, reflect.TypeOf(object))
		} else {
			require.Equal(t, reflect.TypeOf(endpointSlice), reflect.TypeOf(object))
			require.Equal(t, endpointSliceSettings, settings)
		}
	}
	require.Empty(t, wantTypes)
}

func TestCacheByObjectRejectsInvalidWatchLabelSelector(t *testing.T) {
	_, err := cacheByObject("operator.example.com/instance in (", k8sruntime.NewScheme(), meta.NewDefaultRESTMapper(nil))
	require.ErrorContains(t, err, "invalid watch-label-selector")
}
