//go:build e2e

package basicauth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/api/v1alpha1"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/kubeutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	"github.com/kgateway-dev/kgateway/v2/test/e2e"
	testdefaults "github.com/kgateway-dev/kgateway/v2/test/e2e/defaults"
	testmatchers "github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
	"github.com/kgateway-dev/kgateway/v2/test/testutils"
)

var _ e2e.NewSuiteFunc = NewTestingSuite

const (
	// test namespace for proxy resources
	namespace = "default"
	// test service name
	serviceName = "backend-0"
)

var (
	// metadata for gateway - matches the name "super-gateway" from common.yaml
	gateway = &gwv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "super-gateway", Namespace: namespace},
	}
	gatewaytoo = &gwv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "super-gateway-too", Namespace: namespace},
	}

	// metadata for proxy resources
	proxyObjectMeta    = metav1.ObjectMeta{Name: "super-gateway", Namespace: namespace}
	proxyObjectMetaToo = metav1.ObjectMeta{Name: "super-gateway-too", Namespace: namespace}

	proxyDeployment = &appsv1.Deployment{
		ObjectMeta: proxyObjectMeta,
	}
	proxyDeploymentToo = &appsv1.Deployment{
		ObjectMeta: proxyObjectMetaToo,
	}
	proxyService = &corev1.Service{
		ObjectMeta: proxyObjectMeta,
	}
	proxyServiceToo = &corev1.Service{
		ObjectMeta: proxyObjectMetaToo,
	}
	proxyServiceAccount = &corev1.ServiceAccount{
		ObjectMeta: proxyObjectMeta,
	}

	// metadata for backend service
	serviceMeta = metav1.ObjectMeta{
		Namespace: namespace,
		Name:      serviceName,
	}

	simpleSvc = &corev1.Service{
		ObjectMeta: serviceMeta,
	}

	simpleDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceName,
		},
	}

	insecureRoute = &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "route-example-insecure",
		},
	}
	secureGwRoute = &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "route-secure-gw",
		},
	}
	secureGwRouteToo = &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "route-secure-gw-too",
		},
	}
	secureRoute = &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "route-secure",
		},
	}
	secureRouteToo = &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "route-secure-too",
		},
	}
	secureGwPolicy1 = &v1alpha1.AgentgatewayPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "gw-policy-users",
		},
	}
	secureGwPolicy2 = &v1alpha1.AgentgatewayPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "gw-policy-secret",
		},
	}
	secureRoutePolicy1 = &v1alpha1.AgentgatewayPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "secure-route-policy",
		},
	}
	secureRoutePolicy2 = &v1alpha1.AgentgatewayPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "secure-route-with-secret-policy",
		},
	}
	secureGwPolicySecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "basic-auth",
		},
	}
)

// testingSuite is a suite of global rate limiting tests
type testingSuite struct {
	suite.Suite

	ctx context.Context

	// testInstallation contains all the metadata/utilities necessary to execute a series of tests
	// against an installation of kgateway
	testInstallation *e2e.TestInstallation

	// manifests shared by all tests
	commonManifests []string
	// resources from manifests shared by all tests
	commonResources []client.Object
}

func NewTestingSuite(ctx context.Context, testInst *e2e.TestInstallation) suite.TestingSuite {
	return &testingSuite{
		ctx:              ctx,
		testInstallation: testInst,
	}
}

func (s *testingSuite) SetupSuite() {
	s.commonManifests = []string{
		testdefaults.CurlPodManifest,
		getTestFile("common.yaml"),
		getTestFile("insecure-route.yaml"),
		getTestFile("secured-gateway-policy.yaml"),
		getTestFile("secured-route.yaml"),
		getTestFile("service.yaml"),
	}
	s.commonResources = []client.Object{
		// resources from curl manifest
		testdefaults.CurlPod,
		// resources from service manifest
		simpleSvc, simpleDeployment,
		// resources from gateway manifest
		gateway, gatewaytoo,
		// deployer-generated resources
		proxyDeployment, proxyDeploymentToo, proxyService, proxyServiceToo, proxyServiceAccount,
	}

	// set up common resources once
	for _, manifest := range s.commonManifests {
		err := s.testInstallation.Actions.Kubectl().ApplyFile(s.ctx, manifest)
		s.Require().NoError(err, "can apply "+manifest)
	}
	s.testInstallation.Assertions.EventuallyObjectsExist(s.ctx, s.commonResources...)

	// make sure pods are running
	s.testInstallation.Assertions.EventuallyPodsRunning(s.ctx, testdefaults.CurlPod.GetNamespace(), metav1.ListOptions{
		LabelSelector: testdefaults.CurlPodLabelSelector,
	})
	s.testInstallation.Assertions.EventuallyPodsRunning(s.ctx, simpleDeployment.GetNamespace(), metav1.ListOptions{
		LabelSelector: "app=backend-0,version=v1",
	})
	s.testInstallation.Assertions.EventuallyPodsRunning(s.ctx, proxyObjectMeta.GetNamespace(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", testdefaults.WellKnownAppLabel, proxyObjectMeta.GetName()),
	})
}

func (s *testingSuite) TearDownSuite() {
	if testutils.ShouldSkipCleanup(s.T()) {
		return
	}
	// clean up common resources
	for _, manifest := range s.commonManifests {
		err := s.testInstallation.Actions.Kubectl().DeleteFileSafe(s.ctx, manifest)
		s.Require().NoError(err, "can delete "+manifest)
	}
	s.testInstallation.Assertions.EventuallyObjectsNotExist(s.ctx, s.commonResources...)

	// make sure pods are gone
	s.testInstallation.Assertions.EventuallyPodsNotExist(s.ctx, testdefaults.CurlPod.GetNamespace(), metav1.ListOptions{
		LabelSelector: testdefaults.CurlPodLabelSelector,
	})
	s.testInstallation.Assertions.EventuallyPodsNotExist(s.ctx, simpleDeployment.GetNamespace(), metav1.ListOptions{
		LabelSelector: "app=backend-0,version=v1",
	})
	s.testInstallation.Assertions.EventuallyPodsNotExist(s.ctx, proxyObjectMeta.GetNamespace(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", testdefaults.WellKnownAppLabel, proxyObjectMeta.GetName()),
	})
}

func (s *testingSuite) TestRoutePolicy() {
	s.setupTest([]string{}, []client.Object{insecureRoute, secureRoute, secureRouteToo, secureRoutePolicy1, secureRoutePolicy2})

	// TODO (dmitri-d) the below check is failing as there's a gw policy attached to the gw
	//s.assertResponseWithoutAuth(kubeutils.ServiceFQDN(proxyObjectMeta), "insecureroute.com", http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "secureroute.com", base64.StdEncoding.EncodeToString(([]byte)("alice:alicepassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "secureroute.com", base64.StdEncoding.EncodeToString(([]byte)("bob:bobpassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "secureroutetoo.com", base64.StdEncoding.EncodeToString(([]byte)("eve:evepassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "secureroutetoo.com", base64.StdEncoding.EncodeToString(([]byte)("mallory:mallorypassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "secureroute.com", base64.StdEncoding.EncodeToString(([]byte)("trent:book")), http.StatusUnauthorized)
	s.assertResponseWithoutAuth(kubeutils.ServiceFQDN(proxyObjectMeta), "secureroute.com", http.StatusUnauthorized)
}

func (s *testingSuite) TestGatewayPolicy() {
	s.setupTest(nil, []client.Object{secureGwPolicySecret, secureGwRoute, secureGwRouteToo, secureGwPolicy1, secureGwPolicy2})

	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "securegateways.com", base64.StdEncoding.EncodeToString(([]byte)("alice:alicepassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "securegateways.com", base64.StdEncoding.EncodeToString(([]byte)("bob:bobpassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMetaToo), "securegatewaystoo.com", base64.StdEncoding.EncodeToString(([]byte)("eve:evepassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMetaToo), "securegatewaystoo.com", base64.StdEncoding.EncodeToString(([]byte)("mallory:mallorypassword")), http.StatusOK)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMeta), "securegateways.com", base64.StdEncoding.EncodeToString(([]byte)("trent:book")), http.StatusUnauthorized)
	s.assertResponse(kubeutils.ServiceFQDN(proxyObjectMetaToo), "securegatewaystoo.com", base64.StdEncoding.EncodeToString(([]byte)("trent:book")), http.StatusUnauthorized)
	s.assertResponseWithoutAuth(kubeutils.ServiceFQDN(proxyObjectMeta), "securegateways.com", http.StatusUnauthorized)
	s.assertResponseWithoutAuth(kubeutils.ServiceFQDN(proxyObjectMetaToo), "securegatewaystoo.com", http.StatusUnauthorized)
}

func (s *testingSuite) setupTest(manifests []string, resources []client.Object) {
	testutils.Cleanup(s.T(), func() {
		for _, manifest := range manifests {
			err := s.testInstallation.Actions.Kubectl().DeleteFileSafe(s.ctx, manifest)
			s.Require().NoError(err)
		}
		s.testInstallation.Assertions.EventuallyObjectsNotExist(s.ctx, resources...)
	})

	for _, manifest := range manifests {
		err := s.testInstallation.Actions.Kubectl().ApplyFile(s.ctx, manifest)
		s.Require().NoError(err, "can apply "+manifest)
	}
	s.testInstallation.Assertions.EventuallyObjectsExist(s.ctx, resources...)
}

func (s *testingSuite) assertResponse(host, hostHeader, authHeader string, expectedStatus int) {
	s.testInstallation.Assertions.AssertEventualCurlResponse(
		s.ctx,
		testdefaults.CurlPodExecOpt,
		[]curl.Option{
			curl.WithHost(host),
			curl.WithHostHeader(hostHeader),
			curl.WithHeader("Authorization", "Basic "+authHeader),
			curl.WithPort(8080),
		},
		&testmatchers.HttpResponse{
			StatusCode: expectedStatus,
		})
}

func (s *testingSuite) assertResponseWithoutAuth(host, hostHeader string, expectedStatus int) {
	s.testInstallation.Assertions.AssertEventualCurlResponse(
		s.ctx,
		testdefaults.CurlPodExecOpt,
		[]curl.Option{
			curl.WithHost(host),
			curl.WithHostHeader(hostHeader),
			curl.WithPort(8080),
		},
		&testmatchers.HttpResponse{
			StatusCode: expectedStatus,
		})
}

func getTestFile(filename string) string {
	return filepath.Join(fsutils.MustGetThisDir(), "testdata", filename)
}
