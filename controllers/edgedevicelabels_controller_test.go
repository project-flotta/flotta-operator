package controllers_test

import (
	"context"
	"fmt"
	"sort"

	"github.com/golang/mock/gomock"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/controllers"
	"github.com/jakub-dzon/k4e-operator/internal/labels"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("EdgeDeviceLabels controller/Reconcile", func() {
	var (
		mockCtrl                   *gomock.Controller
		deployRepoMock             *edgedeployment.MockRepository
		edgeDeviceRepoMock         *edgedevice.MockRepository
		edgeDeviceLabelsReconciler *controllers.EdgeDeviceLabelsReconciler
		req                        = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test",
				Namespace: "test",
			},
		}
		device *v1alpha1.EdgeDevice
	)

	getDeployment := func(name string) *v1alpha1.EdgeDeployment {
		return &v1alpha1.EdgeDeployment{
			ObjectMeta: v1.ObjectMeta{
				Name:      name,
				Namespace: "test",
			},
			Spec: v1alpha1.EdgeDeploymentSpec{
				Type: "pod",
				Pod:  v1alpha1.Pod{},
				Data: &v1alpha1.DataConfiguration{},
			}}
	}

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		deployRepoMock = edgedeployment.NewMockRepository(mockCtrl)
		edgeDeviceRepoMock = edgedevice.NewMockRepository(mockCtrl)
		edgeDeviceLabelsReconciler = &controllers.EdgeDeviceLabelsReconciler{
			EdgeDeviceRepository:     edgeDeviceRepoMock,
			EdgeDeploymentRepository: deployRepoMock,
		}

		device = &v1alpha1.EdgeDevice{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
			Spec: v1alpha1.EdgeDeviceSpec{
				OsImageId:   "test",
				RequestTime: &v1.Time{},
				Heartbeat:   &v1alpha1.HeartbeatConfiguration{},
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("EdgeDevice deleted", func() {
		// given
		device.DeletionTimestamp = &v1.Time{}
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("EdgeDevice not found", func() {
		// given
		device.DeletionTimestamp = &v1.Time{}
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.NewNotFound(schema.GroupResource{Group: "", Resource: "notfound"}, "notfound")).
			Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("EdgeDevice read failed", func() {
		// given
		device.DeletionTimestamp = &v1.Time{}
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("test")).
			Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).To(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
	})

	It("list EdgeDeployments failed", func() {
		// given
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("test")).
			Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).To(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
	})

	It("invalid deviceSelector", func() {
		// given
		deployment := getDeployment("test")
		deployment.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "test",
					Operator: "dummy",
				},
			},
		}
		deployments := []v1alpha1.EdgeDeployment{*deployment}
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(deployments, nil).
			Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).To(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
	})

	It("list EdgeDeployments returned empty list", func() {
		// given
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, nil).
			Times(2)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("same EdgeDeployment listed multiple times", func() {
		// given
		device.Labels = map[string]string{
			"label1": "",
		}

		deployment := getDeployment("test")
		deployment.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "label1",
					Operator: "Exists",
				},
				{
					Key:      "label2",
					Operator: "DoesNotExist",
				},
			},
		}
		controllers.UpdateSelectorLabels(deployment)
		deploymentList := []v1alpha1.EdgeDeployment{*deployment}

		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel(controllers.DeviceNameLabel), gomock.Any()).
			Return(nil, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(deploymentList, nil).
			Times(2)
		edgeDeviceRepoMock.EXPECT().
			PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
				Expect(edgeDevice.Status.Deployments).To(Equal([]v1alpha1.Deployment{
					{
						Name:  "test",
						Phase: v1alpha1.Deploying,
					},
				}))
			}).Times(1)
		edgeDeviceRepoMock.EXPECT().
			Patch(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, old, new *v1alpha1.EdgeDevice) {
				Expect(new.Labels).To(Equal(map[string]string{
					"label1":                              "",
					labels.WorkloadLabel(deployment.Name): "true",
				}))
			}).Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("no change to device deployments", func() {
		// given
		deployment := getDeployment("test")
		deployment.Spec.Device = device.Name
		controllers.UpdateSelectorLabels(deployment)
		deploymentList := []v1alpha1.EdgeDeployment{*deployment}
		device.Status.Deployments = []v1alpha1.Deployment{
			{
				Name:  "test",
				Phase: v1alpha1.Running,
			},
		}
		addWorkloadLabels(device)

		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel(controllers.DeviceNameLabel), gomock.Any()).
			Return(deploymentList, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, nil).
			Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("new device - test all selector options", func() {
		// given
		deployment1 := getDeployment("test1")
		deployment2 := getDeployment("test2")
		deployment3 := getDeployment("test3")
		deployment4 := getDeployment("test4")
		deployment5 := getDeployment("test5")
		deployment1.Spec.Device = device.Name
		deployment2.Spec.DeviceSelector = &v1.LabelSelector{
			MatchLabels: map[string]string{"label1": "label1"},
		}
		deployment3.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "notexist",
					Operator: v1.LabelSelectorOpDoesNotExist,
				},
			},
		}
		deployment4.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "exist",
					Operator: v1.LabelSelectorOpExists,
				},
			},
		}
		deployment5.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "label2",
					Operator: v1.LabelSelectorOpIn,
					Values:   []string{"dummy1", "label2", "dummy2"},
				},
			},
		}
		controllers.UpdateSelectorLabels(deployment1, deployment2, deployment3, deployment4, deployment5)

		device.Labels = map[string]string{
			"label1": "label1",
			"exist":  "dada",
			"label2": "label2",
		}

		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel(controllers.DeviceNameLabel), gomock.Any()).
			Return([]v1alpha1.EdgeDeployment{*deployment1}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel("label1"), gomock.Any()).
			Return([]v1alpha1.EdgeDeployment{*deployment2}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel(controllers.DoesNotExistLabel), gomock.Any()).
			Return([]v1alpha1.EdgeDeployment{*deployment3}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel("exist"), gomock.Any()).
			Return([]v1alpha1.EdgeDeployment{*deployment4}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel("label2"), gomock.Any()).
			Return([]v1alpha1.EdgeDeployment{*deployment5}, nil).
			Times(1)
		edgeDeviceRepoMock.EXPECT().
			PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
				Expect(sortDeployments(edgeDevice.Status.Deployments)).To(Equal([]v1alpha1.Deployment{
					{Name: "test1", Phase: v1alpha1.Deploying},
					{Name: "test2", Phase: v1alpha1.Deploying},
					{Name: "test3", Phase: v1alpha1.Deploying},
					{Name: "test4", Phase: v1alpha1.Deploying},
					{Name: "test5", Phase: v1alpha1.Deploying},
				}))
			}).Times(1)
		edgeDeviceRepoMock.EXPECT().
			Patch(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, old, new *v1alpha1.EdgeDevice) {
				Expect(new.Labels).To(Equal(map[string]string{
					"label1":                      "label1",
					"label2":                      "label2",
					"exist":                       "dada",
					labels.WorkloadLabel("test1"): "true",
					labels.WorkloadLabel("test2"): "true",
					labels.WorkloadLabel("test3"): "true",
					labels.WorkloadLabel("test4"): "true",
					labels.WorkloadLabel("test5"): "true",
				}))
			}).Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("device labels changed", func() {
		// given
		device.Status.Deployments = []v1alpha1.Deployment{
			{
				Name:  "toremove",
				Phase: v1alpha1.Running,
			},
			{
				Name:  "tokeep",
				Phase: v1alpha1.Running,
			},
		}
		addWorkloadLabels(device)
		device.Labels["tokeep"] = "tokeep"
		device.Labels["toadd"] = "toadd"

		deploymentToAdd := getDeployment("toadd")
		deploymentToKeep := getDeployment("tokeep")
		deploymentToKeep.Spec.DeviceSelector = &v1.LabelSelector{
			MatchLabels: map[string]string{"tokeep": "tokeep"},
		}
		deploymentToAdd.Spec.DeviceSelector = &v1.LabelSelector{
			MatchLabels: map[string]string{"toadd": "toadd"},
		}
		controllers.UpdateSelectorLabels(deploymentToAdd, deploymentToKeep)

		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel("toadd"), gomock.Any()).
			Return([]v1alpha1.EdgeDeployment{*deploymentToAdd}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), controllers.CreateSelectorLabel("tokeep"), gomock.Any()).
			Return([]v1alpha1.EdgeDeployment{*deploymentToKeep}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, nil).
			Times(2)

		edgeDeviceRepoMock.EXPECT().
			PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
				Expect(sortDeployments(edgeDevice.Status.Deployments)).To(Equal([]v1alpha1.Deployment{
					{Name: "toadd", Phase: v1alpha1.Deploying},
					{Name: "tokeep", Phase: v1alpha1.Running},
				}))
			}).Times(1)
		edgeDeviceRepoMock.EXPECT().
			Patch(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, old, new *v1alpha1.EdgeDevice) {
				Expect(new.Labels).To(Equal(map[string]string{
					"tokeep":                       "tokeep",
					"toadd":                        "toadd",
					labels.WorkloadLabel("tokeep"): "true",
					labels.WorkloadLabel("toadd"):  "true",
				}))
			}).Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

})

func sortDeployments(deployments []v1alpha1.Deployment) []v1alpha1.Deployment {
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].Name < deployments[j].Name
	})
	return deployments
}

func addWorkloadLabels(device *v1alpha1.EdgeDevice) {
	if len(device.Status.Deployments) == 0 {
		return
	}

	if device.Labels == nil {
		device.Labels = map[string]string{}
	}

	for _, deployment := range device.Status.Deployments {
		device.Labels[labels.WorkloadLabel(deployment.Name)] = "true"
	}
}
