package devicemetrics_test

import (
	"context"
	"github.com/go-openapi/errors"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/project-flotta/flotta-operator/internal/devicemetrics"
	"github.com/project-flotta/flotta-operator/internal/k8sclient"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mapName      = "allow-list-map"
	mapNamespace = "any-namespace"

	metricsListYaml = `
names:
  - cpu_cores
  - free_mem
`
)

var _ = Describe("Device Metrics", func() {
	var (
		mockCtrl  *gomock.Controller
		k8sClient *k8sclient.MockK8sClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		k8sClient = k8sclient.NewMockK8sClient(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("System Allow Lists", func() {
		var (
			alGenerator devicemetrics.AllowListGenerator
		)

		BeforeEach(func() {
			alGenerator = devicemetrics.NewAllowListGenerator(k8sClient)
		})

		It("should load existing allow list", func() {
			// given
			k8sClient.EXPECT().Get(
				gomock.AssignableToTypeOf(context.TODO()),
				gomock.Eq(client.ObjectKey{Name: mapName, Namespace: mapNamespace}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				DoAndReturn(configMapGenerator(mapName, mapNamespace, map[string]string{"metrics_list.yaml": metricsListYaml}))

			// when
			allowList, err := alGenerator.GenerateFromConfigMap(context.TODO(), mapName, mapNamespace)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(allowList).ToNot(BeNil())
			Expect(allowList.Names).To(ConsistOf("cpu_cores", "free_mem"))
		})

		It("should fail when allow list config map is missing", func() {
			// given
			k8sClient.EXPECT().Get(
				gomock.AssignableToTypeOf(context.TODO()),
				gomock.Eq(client.ObjectKey{Name: mapName, Namespace: mapNamespace}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				Return(errors.NotFound("Not found"))

			// when
			_, err := alGenerator.GenerateFromConfigMap(context.TODO(), mapName, mapNamespace)

			// then
			Expect(err).To(HaveOccurred())
		})

		It("should fail when allow list data yaml is missing", func() {
			// given
			k8sClient.EXPECT().Get(
				gomock.AssignableToTypeOf(context.TODO()),
				gomock.Eq(client.ObjectKey{Name: mapName, Namespace: mapNamespace}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				DoAndReturn(configMapGenerator(mapName, mapNamespace, map[string]string{}))

			// when
			_, err := alGenerator.GenerateFromConfigMap(context.TODO(), mapName, mapNamespace)

			// then
			Expect(err).To(HaveOccurred())
		})

		It("should fail when allow list data yaml is incorrect", func() {
			// given
			k8sClient.EXPECT().Get(
				gomock.AssignableToTypeOf(context.TODO()),
				gomock.Eq(client.ObjectKey{Name: mapName, Namespace: mapNamespace}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				DoAndReturn(configMapGenerator(mapName, mapNamespace, map[string]string{"metrics_list.yaml": "that's not YAML"}))

			// when
			_, err := alGenerator.GenerateFromConfigMap(context.TODO(), mapName, mapNamespace)

			// then
			Expect(err).To(HaveOccurred())
		})
	})
})

func configMapGenerator(name, namespace string, data map[string]string) func(_ context.Context, _ client.ObjectKey, obj client.Object) error {
	return func(_ context.Context, _ client.ObjectKey, obj client.Object) error {
		cm := obj.(*corev1.ConfigMap)
		cm.SetName(name)
		cm.SetNamespace(namespace)
		cm.Data = data
		return nil
	}
}
