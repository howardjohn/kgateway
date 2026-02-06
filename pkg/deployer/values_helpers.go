package deployer

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strings"

	"istio.io/istio/pkg/slices"
	"istio.io/istio/pkg/util/smallset"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/kgateway/validate"
)

var (
	// ErrMultipleAddresses is returned when multiple addresses are specified in Gateway.spec.addresses
	ErrMultipleAddresses = errors.New("multiple addresses given, only one address is supported")

	// ErrNoValidIPAddress is returned when no valid IP address is found in Gateway.spec.addresses
	ErrNoValidIPAddress = errors.New("IP address in Gateway.spec.addresses not valid")
)

// This file contains helper functions that generate helm values in the format needed
// by the deployer.

var ComponentLogLevelEmptyError = func(key string, value string) error {
	return fmt.Errorf("an empty key or value was provided in componentLogLevels: key=%s, value=%s", key, value)
}


// TODODONOTMERGE
type GatewayForDeployer struct {
	ObjectSource
	// Controller name for the gateway
	ControllerName string
	// All ports from all listeners
	Ports smallset.Set[int32]
}

type ObjectSource struct {
	Group     string `json:"group,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// GetKind returns the kind of the route.
func (c ObjectSource) GetGroupKind() schema.GroupKind {
	return schema.GroupKind{
		Group: c.Group,
		Kind:  c.Kind,
	}
}

// GetName returns the name of the route.
func (c ObjectSource) GetName() string {
	return c.Name
}

// GetNamespace returns the namespace of the route.
func (c ObjectSource) GetNamespace() string {
	return c.Namespace
}

func (c ObjectSource) ResourceName() string {
	return fmt.Sprintf("%s/%s/%s/%s", c.Group, c.Kind, c.Namespace, c.Name)
}

func (c ObjectSource) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", c.Group, c.Kind, c.Namespace, c.Name)
}

func (c ObjectSource) Equals(in ObjectSource) bool {
	return c.Namespace == in.Namespace && c.Name == in.Name && c.Group == in.Group && c.Kind == in.Kind
}

func (c ObjectSource) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}
func (c GatewayForDeployer) ResourceName() string {
	return c.ObjectSource.ResourceName()
}

func (c GatewayForDeployer) Equals(in GatewayForDeployer) bool {
	return c.ObjectSource.Equals(in.ObjectSource) &&
		c.ControllerName == in.ControllerName &&
		slices.Equal(c.Ports.List(), in.Ports.List())
}
// Extract the listener ports from a Gateway and corresponding listener sets. These will be used to populate:
// 1. the ports exposed on the envoy container
// 2. the ports exposed on the proxy service
func GetPortsValues(gw *GatewayForDeployer, agentgateway bool) []HelmPort {
	gwPorts := []HelmPort{}

	// Add ports from Gateway listeners
	for _, port := range gw.Ports.List() {
		portName := GenerateListenerNameFromPort(port)
		if err := validate.ListenerPortForParent(port, agentgateway); err != nil {
			// skip invalid ports; statuses are handled in the translator
			logger.Error("skipping port", "gateway", gw.ResourceName(), "error", err)
			continue
		}
		gwPorts = AppendPortValue(gwPorts, port, portName)
	}

	return gwPorts
}

func SanitizePortName(name string) string {
	nonAlphanumericRegex := regexp.MustCompile(`[^a-zA-Z0-9-]+`)
	str := nonAlphanumericRegex.ReplaceAllString(name, "-")
	doubleHyphen := regexp.MustCompile(`-{2,}`)
	str = doubleHyphen.ReplaceAllString(str, "-")

	// This is a kubernetes spec requirement.
	maxPortNameLength := 15
	if len(str) > maxPortNameLength {
		str = str[:maxPortNameLength]
	}
	return str
}

func AppendPortValue(gwPorts []HelmPort, port int32, name string) []HelmPort {
	if slices.IndexFunc(gwPorts, func(p HelmPort) bool { return *p.Port == port }) != -1 {
		return gwPorts
	}

	portName := SanitizePortName(name)
	protocol := "TCP"

	return append(gwPorts, HelmPort{
		Port:       &port,
		TargetPort: &port,
		Name:       &portName,
		Protocol:   &protocol,
	})
}

// GetLoadBalancerIPFromGatewayAddresses extracts the IP address from Gateway.spec.addresses.
// Returns the IP address if exactly one valid IP address is found, nil if no addresses are specified,
// or an error if more than one address is specified or no valid IP address is found.
func GetLoadBalancerIPFromGatewayAddresses(gw *gwv1.Gateway) (*string, error) {
	ipAddresses := slices.MapFilter(gw.Spec.Addresses, func(addr gwv1.GatewaySpecAddress) *string {
		if addr.Type == nil || *addr.Type == gwv1.IPAddressType {
			return &addr.Value
		}
		return nil
	})

	if len(ipAddresses) == 0 && len(gw.Spec.Addresses) != 0 {
		return nil, ErrNoValidIPAddress
	}

	if len(ipAddresses) == 0 {
		return nil, nil
	}
	if len(ipAddresses) > 1 {
		return nil, fmt.Errorf("%w: gateway %s/%s has %d addresses", ErrMultipleAddresses, gw.Namespace, gw.Name, len(gw.Spec.Addresses))
	}

	addr := ipAddresses[0]

	// Validate IP format
	parsedIP, err := netip.ParseAddr(addr)
	if err == nil && parsedIP.IsValid() {
		return &addr, nil
	}
	return nil, ErrNoValidIPAddress
}


// SetLoadBalancerIPFromGatewayForAgentgateway extracts the IP address from Gateway.spec.addresses
// and sets it on the AgentgatewayHelmService.
// Only sets the IP if exactly one valid IP address is found in Gateway.spec.addresses.
// Returns an error if more than one address is specified or no valid IP address is found.
// Note: Agentgateway services are always LoadBalancer type, so no service type check is needed.
func SetLoadBalancerIPFromGatewayForAgentgateway(gw *gwv1.Gateway, svc *AgentgatewayHelmService) error {
	ip, err := GetLoadBalancerIPFromGatewayAddresses(gw)
	if err != nil {
		return err
	}
	if ip != nil {
		svc.LoadBalancerIP = ip
	}
	return nil
}

// ComponentLogLevelsToString converts the key-value pairs in the map into a string of the
// format: key1:value1,key2:value2,key3:value3, where the keys are sorted alphabetically.
// If an empty map is passed in, then an empty string is returned.
// Map keys and values may not be empty.
// No other validation is currently done on the keys/values.
func ComponentLogLevelsToString(vals map[string]string) (string, error) {
	if len(vals) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(vals))
	for k, v := range vals {
		if k == "" || v == "" {
			return "", ComponentLogLevelEmptyError(k, v)
		}
		parts = append(parts, fmt.Sprintf("%s:%s", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ","), nil
}

func GenerateListenerNameFromPort(port gwv1.PortNumber) string {
	// Add a ~ to make sure the name won't collide with user provided names in other listeners
	return fmt.Sprintf("listener~%d", port)
}
