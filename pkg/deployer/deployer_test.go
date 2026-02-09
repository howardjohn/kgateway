package deployer_test

import (
	"context"
	"errors"

	envoybootstrapv3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	"github.com/ghodss/yaml"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"istio.io/istio/pkg/config/schema/gvk"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	agentgatewayv1alpha1 "github.com/kgateway-dev/kgateway/v2/api/v1alpha1/agentgateway"
	"github.com/kgateway-dev/kgateway/v2/pkg/apiclient"
	"github.com/kgateway-dev/kgateway/v2/pkg/apiclient/fake"
	"github.com/kgateway-dev/kgateway/v2/pkg/deployer"
	deployerinternal "github.com/kgateway-dev/kgateway/v2/pkg/kgateway/deployer"
	"github.com/kgateway-dev/kgateway/v2/pkg/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/schemes"
	deployertest "github.com/kgateway-dev/kgateway/v2/test/deployer"
)

const (
	defaultNamespace = "default"
	envoyDataKey     = "envoy.yaml"
)

var scheme = schemes.DefaultScheme()

func unmarshalYaml(data []byte, into proto.Message) error {
	jsn, err := yaml.YAMLToJSON(data)
	if err != nil {
		return err
	}

	var j jsonpb.UnmarshalOptions

	return j.Unmarshal(jsn, into)
}

type clientObjects []client.Object

func (objs *clientObjects) findDeployment(name string) *appsv1.Deployment {
	for _, obj := range *objs {
		if dep, ok := obj.(*appsv1.Deployment); ok {
			if dep.Name == name && dep.Namespace == defaultNamespace {
				return dep
			}
		}
	}
	return nil
}

func (objs *clientObjects) findServiceAccount(name string) *corev1.ServiceAccount {
	for _, obj := range *objs {
		if sa, ok := obj.(*corev1.ServiceAccount); ok {
			if sa.Name == name && sa.Namespace == defaultNamespace {
				return sa
			}
		}
	}
	return nil
}

func (objs *clientObjects) findService(name string) *corev1.Service {
	for _, obj := range *objs {
		if svc, ok := obj.(*corev1.Service); ok {
			if svc.Name == name && svc.Namespace == defaultNamespace {
				return svc
			}
		}
	}
	return nil
}

func (objs *clientObjects) findConfigMap(namespace, name string) *corev1.ConfigMap {
	for _, obj := range *objs {
		if cm, ok := obj.(*corev1.ConfigMap); ok {
			if cm.Name == name && cm.Namespace == namespace {
				return cm
			}
		}
	}
	return nil
}

func (objs *clientObjects) getEnvoyConfig(namespace, name string) *envoybootstrapv3.Bootstrap {
	cm := objs.findConfigMap(namespace, name).Data
	var bootstrapCfg envoybootstrapv3.Bootstrap
	err := unmarshalYaml([]byte(cm[envoyDataKey]), &bootstrapCfg)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return &bootstrapCfg
}

var _ = Describe("Deployer", func() {
	var (
		agentgatewayParam = func(name string) *agentgatewayv1alpha1.AgentgatewayParameters {
			return &agentgatewayv1alpha1.AgentgatewayParameters{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: defaultNamespace,
					UID:       "1237",
				},
				Spec: agentgatewayv1alpha1.AgentgatewayParametersSpec{
					AgentgatewayParametersConfigs: agentgatewayv1alpha1.AgentgatewayParametersConfigs{
						Image: &agentgatewayv1alpha1.Image{
							Repository: ptr.To("agentgateway"),
							Tag:        ptr.To("0.4.0"),
						},
						Resources: &corev1.ResourceRequirements{
							Limits: corev1.ResourceList{"cpu": resource.MustParse("101m")},
						},
						Env: []corev1.EnvVar{
							{
								Name:  "test",
								Value: "value",
							},
						},
					},
				},
			}
		}
	)

	Context("agentgateway", func() {
		var (
			agwp *agentgatewayv1alpha1.AgentgatewayParameters
			gwc  *gwv1.GatewayClass
		)
		BeforeEach(func() {
			agwp = agentgatewayParam("agent-gateway-params")
			gwc = &gwv1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: wellknown.DefaultAgwClassName,
				},
				Spec: gwv1.GatewayClassSpec{
					ControllerName: wellknown.DefaultAgwControllerName,
					ParametersRef: &gwv1.ParametersReference{
						Group:     agentgatewayv1alpha1.GroupName,
						Kind:      gwv1.Kind(wellknown.AgentgatewayParametersGVK.Kind),
						Name:      agwp.GetName(),
						Namespace: ptr.To(gwv1.Namespace(defaultNamespace)),
					},
				},
			}
		})

		It("deploys agentgateway", func() {
			gw := &gwv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "agent-gateway",
					Namespace: defaultNamespace,
				},
				Spec: gwv1.GatewaySpec{
					GatewayClassName: wellknown.DefaultAgwClassName,
					Infrastructure: &gwv1.GatewayInfrastructure{
						ParametersRef: &gwv1.LocalParametersReference{
							Group: agentgatewayv1alpha1.GroupName,
							Kind:  gwv1.Kind(wellknown.AgentgatewayParametersGVK.Kind),
							Name:  agwp.GetName(),
						},
					},
					Listeners: []gwv1.Listener{{
						Name: "listener-1",
						Port: 80,
					}},
				},
			}
			fakeClient := fake.NewClient(GinkgoT(), gwc, agwp)
			gwParams := deployerinternal.NewGatewayParameters(fakeClient, &deployer.Inputs{
				CommonCollections: deployertest.NewCommonCols(),
				Dev:               false,
				ControlPlane: deployer.ControlPlaneInfo{
					XdsHost:    "something.cluster.local",
					AgwXdsPort: 5678,
				},
				AgentgatewayClassName:      wellknown.DefaultAgwClassName,
				AgentgatewayControllerName: wellknown.DefaultAgwControllerName,
			})
			d, err := deployerinternal.NewGatewayDeployer(
				wellknown.DefaultAgwControllerName,
				wellknown.DefaultAgwClassName,
				scheme,
				fakeClient,
				gwParams,
			)
			Expect(err).NotTo(HaveOccurred())
			fakeClient.RunAndWait(context.Background().Done())

			var objs clientObjects
			objs, err = d.GetObjsToDeploy(context.Background(), gw)
			Expect(err).NotTo(HaveOccurred())
			objs = d.SetNamespaceAndOwner(gw, objs)
			// check the image is using the agentgateway image
			deployment := objs.findDeployment("agent-gateway")
			Expect(deployment).ToNot(BeNil())
			// check the image uses the override tag
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(ContainSubstring("agentgateway"))
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(ContainSubstring("0.4.0"))
			// check resource requirements are correctly set
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().Equal(resource.MustParse("101m"))).To(BeTrue())
			// check env values are appended to the end of the list
			var testEnvVar corev1.EnvVar
			for _, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
				if envVar.Name == "test" {
					testEnvVar = envVar
					break
				}
			}
			Expect(testEnvVar.Name).To(Equal("test"))
			Expect(testEnvVar.Value).To(Equal("value"))
			// check the service is using the agentgateway port
			svc := objs.findService("agent-gateway")
			Expect(svc).ToNot(BeNil())
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(80)))
			// check the config map is using the xds address and port
			cm := objs.findConfigMap(defaultNamespace, "agent-gateway")
			Expect(cm).ToNot(BeNil())
		})
	})
})

var _ = Describe("DeployObjs", func() {
	var (
		ns   = "test-ns"
		name = "test-obj"
		ctx  = context.Background()
	)

	getDeployer := func(fc apiclient.Client, patcher deployer.Patcher) *deployer.Deployer {
		d, err := deployerinternal.NewGatewayDeployer(
			wellknown.DefaultAgwControllerName,
			wellknown.DefaultAgwClassName,
			scheme,
			fc,
			nil,
			deployer.WithPatcher(patcher),
		)
		Expect(err).ToNot(HaveOccurred())
		return d
	}

	It("skips patch if object is unchanged", func() {
		cm := &corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{Kind: gvk.ConfigMap.Kind, APIVersion: gvk.ConfigMap.GroupVersion()},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Data:       map[string]string{"foo": "bar"},
		}
		fc := fake.NewClient(GinkgoT(), cm.DeepCopy())
		d := getDeployer(fc, func(client apiclient.Client, fieldManager string, gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error {
			Fail("Patch should not be called")
			return errors.New("unexpected Patch call")
		})
		fc.RunAndWait(context.Background().Done())

		err := d.DeployObjs(ctx, []client.Object{cm})
		Expect(err).ToNot(HaveOccurred())
	})

	It("skips patch when only change is object status", func() {
		pod1 := &corev1.Pod{
			TypeMeta:   metav1.TypeMeta{Kind: gvk.Pod.Kind, APIVersion: gvk.Pod.GroupVersion()},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "test", Image: "test:latest"}}},
			Status:     corev1.PodStatus{Phase: corev1.PodPending},
		}
		pod2 := pod1.DeepCopy()

		// obj to deploy won't have a status set.
		pod2.Status = corev1.PodStatus{}
		fc := fake.NewClient(GinkgoT(), pod1.DeepCopy())
		d := getDeployer(fc, func(client apiclient.Client, fieldManager string, gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error {
			Fail("Patch should not be called")
			return errors.New("unexpected Patch call")
		})
		fc.RunAndWait(context.Background().Done())

		err := d.DeployObjs(ctx, []client.Object{pod2})
		Expect(err).ToNot(HaveOccurred())
	})

	It("patches if object is different", func() {
		cm := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{Kind: gvk.ConfigMap.Kind, APIVersion: gvk.ConfigMap.GroupVersion()},

			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Data:       map[string]string{"foo": "bar"},
		}
		fc := fake.NewClient(GinkgoT(), cm.DeepCopy())
		cm.Data = map[string]string{"foo": "bar", "bar": "baz"}
		patched := false
		d := getDeployer(fc, func(client apiclient.Client, fieldManager string, gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error {
			patched = true
			return nil
		})
		fc.RunAndWait(context.Background().Done())

		err := d.DeployObjs(ctx, []client.Object{cm})
		Expect(err).ToNot(HaveOccurred())
		Expect(patched).To(BeTrue())
	})

	It("patches if object does not exist (IsNotFound error)", func() {
		cm := &corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{Kind: gvk.ConfigMap.Kind, APIVersion: gvk.ConfigMap.GroupVersion()},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		}
		fc := fake.NewClient(GinkgoT())
		patched := false
		d := getDeployer(fc, func(client apiclient.Client, fieldManager string, gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error {
			patched = true
			return nil
		})
		fc.RunAndWait(context.Background().Done())

		err := d.DeployObjs(ctx, []client.Object{cm})
		Expect(err).ToNot(HaveOccurred())
		Expect(patched).To(BeTrue())
	})

	It("uses GatewayClass controllerName (not class name) as SSA field manager", func() {
		customClassName := "custom-agw-class"
		gwc := &gwv1.GatewayClass{
			ObjectMeta: metav1.ObjectMeta{Name: customClassName},
			Spec:       gwv1.GatewayClassSpec{ControllerName: wellknown.DefaultAgwControllerName},
		}
		gw := &gwv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Name: "test-gw", Namespace: ns, UID: "12345"},
			Spec:       gwv1.GatewaySpec{GatewayClassName: gwv1.ObjectName(customClassName)},
		}
		gw.SetGroupVersionKind(wellknown.GatewayGVK)
		cm := &corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{Kind: gvk.ConfigMap.Kind, APIVersion: gvk.ConfigMap.GroupVersion()},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Data:       map[string]string{"foo": "bar"},
		}

		fc := fake.NewClient(GinkgoT(), gwc)
		var usedFieldManager string
		d := getDeployer(fc, func(client apiclient.Client, fieldManager string, gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error {
			usedFieldManager = fieldManager
			return nil
		})
		fc.RunAndWait(context.Background().Done())

		err := d.DeployObjsWithSource(ctx, []client.Object{cm}, gw)
		Expect(err).ToNot(HaveOccurred())
		Expect(usedFieldManager).To(Equal(wellknown.DefaultAgwControllerName))
	})

	It("falls back to class name comparison when GatewayClass lookup fails", func() {
		gw := &gwv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Name: "test-gw", Namespace: ns, UID: "12345"},
			Spec:       gwv1.GatewaySpec{GatewayClassName: wellknown.DefaultAgwClassName},
		}
		gw.SetGroupVersionKind(wellknown.GatewayGVK)
		cm := &corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{Kind: gvk.ConfigMap.Kind, APIVersion: gvk.ConfigMap.GroupVersion()},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Data:       map[string]string{"foo": "bar"},
		}

		fc := fake.NewClient(GinkgoT()) // no GatewayClass created
		var usedFieldManager string
		d := getDeployer(fc, func(client apiclient.Client, fieldManager string, gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error {
			usedFieldManager = fieldManager
			return nil
		})
		fc.RunAndWait(context.Background().Done())

		err := d.DeployObjsWithSource(ctx, []client.Object{cm}, gw)
		Expect(err).ToNot(HaveOccurred())
		Expect(usedFieldManager).To(Equal(wellknown.DefaultAgwControllerName))
	})
})
