//go:build conformance

package conformance_test

import (
	"context"
	"fmt"
	"testing"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/gateway-api/conformance"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"
)

func TestConformance(t *testing.T) {
	options := conformance.DefaultOptions(t)

	// Auto-detect the Gateway API channel by checking installed CRDs
	channel, err := detectGatewayAPIChannel()
	if err != nil {
		t.Logf("Failed to detect Gateway API channel, defaulting to experimental: %v", err)
		channel = features.FeatureChannelExperimental
	} else {
		t.Logf("Detected Gateway API channel: %s", channel)
	}

	// Configure profiles and exempt features based on detected channel
	profiles := sets.New(suite.GatewayGRPCConformanceProfileName, suite.GatewayHTTPConformanceProfileName)
	if channel == features.FeatureChannelExperimental {
		profiles.Insert(suite.GatewayTLSConformanceProfileName)
	}
	options.ConformanceProfiles = profiles
	options.SkipTests = []string{}
	if channel == features.FeatureChannelStandard {
		exemptExperimentalFeatures(&options)
	}

	t.Logf("Running conformance tests with\nprofiles: %+v\n", profiles)
	conformance.RunConformanceWithOptions(t, options)
}

func exemptExperimentalFeatures(options *suite.ConformanceOptions) {
	if options.ExemptFeatures == nil {
		options.ExemptFeatures = suite.FeaturesSet{}
	}
	for _, feature := range features.AllFeatures.UnsortedList() {
		if feature.Channel == features.FeatureChannelExperimental {
			options.ExemptFeatures.Insert(feature.Name)
		}
	}
}

// detectGatewayAPIChannel checks which Gateway API CRDs are installed to determine the channel
func detectGatewayAPIChannel() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	clientset, err := apiextensionsclient.NewForConfig(cfg)
	if err != nil {
		return "", err
	}

	// Check the gateway.networking.k8s.io/channel annotation on HTTPRoute CRD
	crd, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Get(
		context.Background(),
		"httproutes.gateway.networking.k8s.io",
		metav1.GetOptions{},
	)
	if err != nil {
		return "", err
	}

	channel := crd.Annotations["gateway.networking.k8s.io/channel"]
	if channel == "" {
		return "", fmt.Errorf("gateway.networking.k8s.io/channel annotation not found on HTTPRoute CRD")
	}

	return channel, nil
}
