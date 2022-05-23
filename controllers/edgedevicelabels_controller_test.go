package controllers_test

import (
	"context"
	"fmt"
	"github.com/project-flotta/flotta-operator/internal/common/labels"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeworkload"
	"sort"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/controllers"
)

var _ = Describe("EdgeDeviceLabels controller/Reconcile", func() {
	const (
		namespace string = "test"
	)
	var (
		mockCtrl                   *gomock.Controller
		deployRepoMock             *edgeworkload.MockRepository
		edgeDeviceRepoMock         *edgedevice.MockRepository
		edgeDeviceLabelsReconciler *controllers.EdgeDeviceLabelsReconciler
		req                        = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test",
				Namespace: namespace,
			},
		}
		device *v1alpha1.EdgeDevice
	)

	getWorkload := func(name string) *v1alpha1.EdgeWorkload {
		return &v1alpha1.EdgeWorkload{
			ObjectMeta: v1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha1.EdgeWorkloadSpec{
				Type: "pod",
				Pod:  v1alpha1.Pod{},
				Data: &v1alpha1.DataConfiguration{},
			}}
	}

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		deployRepoMock = edgeworkload.NewMockRepository(mockCtrl)
		edgeDeviceRepoMock = edgedevice.NewMockRepository(mockCtrl)
		edgeDeviceLabelsReconciler = &controllers.EdgeDeviceLabelsReconciler{
			EdgeDeviceRepository:   edgeDeviceRepoMock,
			EdgeWorkloadRepository: deployRepoMock,
		}

		device = &v1alpha1.EdgeDevice{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: namespace,
			},
			Spec: v1alpha1.EdgeDeviceSpec{
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

	It("list EdgeWorkloads failed", func() {
		// given
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any(), namespace).
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
		workload := getWorkload("test")
		workload.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "test",
					Operator: "dummy",
				},
			},
		}
		workloads := []v1alpha1.EdgeWorkload{*workload}
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any(), namespace).
			Return(workloads, nil).
			Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).To(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
	})

	It("list EdgeWorkloads returned empty list", func() {
		// given
		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any(), namespace).
			Return(nil, nil).
			Times(2)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("same EdgeWorkload listed multiple times", func() {
		// given
		device.Labels = map[string]string{
			"label1": "",
		}

		workload := getWorkload("test")
		workload.Spec.DeviceSelector = &v1.LabelSelector{
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
		controllers.UpdateSelectorLabels(workload)
		workloadList := []v1alpha1.EdgeWorkload{*workload}

		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel(labels.DeviceNameLabel), gomock.Any(), namespace).
			Return(nil, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any(), namespace).
			Return(workloadList, nil).
			Times(2)
		edgeDeviceRepoMock.EXPECT().
			PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
				Expect(edgeDevice.Status.Workloads).To(Equal([]v1alpha1.Workload{
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
					"label1":                            "",
					labels.WorkloadLabel(workload.Name): "true",
				}))
			}).Times(1)

		// when
		res, err := edgeDeviceLabelsReconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
	})

	It("no change to device workloads", func() {
		// given
		workload := getWorkload("test")
		workload.Spec.Device = device.Name
		controllers.UpdateSelectorLabels(workload)
		workloadList := []v1alpha1.EdgeWorkload{*workload}
		device.Status.Workloads = []v1alpha1.Workload{
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
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel(labels.DeviceNameLabel), gomock.Any(), namespace).
			Return(workloadList, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any(), namespace).
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
		workload1 := getWorkload("test1")
		workload2 := getWorkload("test2")
		workload3 := getWorkload("test3")
		workload4 := getWorkload("test4")
		workload5 := getWorkload("test5")
		workload1.Spec.Device = device.Name
		workload2.Spec.DeviceSelector = &v1.LabelSelector{
			MatchLabels: map[string]string{"label1": "label1"},
		}
		workload3.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "notexist",
					Operator: v1.LabelSelectorOpDoesNotExist,
				},
			},
		}
		workload4.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "exist",
					Operator: v1.LabelSelectorOpExists,
				},
			},
		}
		workload5.Spec.DeviceSelector = &v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key:      "label2",
					Operator: v1.LabelSelectorOpIn,
					Values:   []string{"dummy1", "label2", "dummy2"},
				},
			},
		}
		controllers.UpdateSelectorLabels(workload1, workload2, workload3, workload4, workload5)

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
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel(labels.DeviceNameLabel), gomock.Any(), namespace).
			Return([]v1alpha1.EdgeWorkload{*workload1}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel("label1"), gomock.Any(), namespace).
			Return([]v1alpha1.EdgeWorkload{*workload2}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel(labels.DoesNotExistLabel), gomock.Any(), namespace).
			Return([]v1alpha1.EdgeWorkload{*workload3}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel("exist"), gomock.Any(), namespace).
			Return([]v1alpha1.EdgeWorkload{*workload4}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel("label2"), gomock.Any(), namespace).
			Return([]v1alpha1.EdgeWorkload{*workload5}, nil).
			Times(1)
		edgeDeviceRepoMock.EXPECT().
			PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
				Expect(sortWorkloads(edgeDevice.Status.Workloads)).To(Equal([]v1alpha1.Workload{
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
		device.Status.Workloads = []v1alpha1.Workload{
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

		workloadToAdd := getWorkload("toadd")
		workloadToKeep := getWorkload("tokeep")
		workloadToKeep.Spec.DeviceSelector = &v1.LabelSelector{
			MatchLabels: map[string]string{"tokeep": "tokeep"},
		}
		workloadToAdd.Spec.DeviceSelector = &v1.LabelSelector{
			MatchLabels: map[string]string{"toadd": "toadd"},
		}
		controllers.UpdateSelectorLabels(workloadToAdd, workloadToKeep)

		edgeDeviceRepoMock.EXPECT().
			Read(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(device, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel("toadd"), gomock.Any(), namespace).
			Return([]v1alpha1.EdgeWorkload{*workloadToAdd}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), labels.CreateSelectorLabel("tokeep"), gomock.Any(), namespace).
			Return([]v1alpha1.EdgeWorkload{*workloadToKeep}, nil).
			Times(1)
		deployRepoMock.EXPECT().
			ListByLabel(gomock.Any(), gomock.Any(), gomock.Any(), namespace).
			Return(nil, nil).
			Times(2)

		edgeDeviceRepoMock.EXPECT().
			PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
				Expect(sortWorkloads(edgeDevice.Status.Workloads)).To(Equal([]v1alpha1.Workload{
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

func sortWorkloads(workloads []v1alpha1.Workload) []v1alpha1.Workload {
	sort.Slice(workloads, func(i, j int) bool {
		return workloads[i].Name < workloads[j].Name
	})
	return workloads
}

func addWorkloadLabels(device *v1alpha1.EdgeDevice) {
	if len(device.Status.Workloads) == 0 {
		return
	}

	if device.Labels == nil {
		device.Labels = map[string]string{}
	}

	for _, workload := range device.Status.Workloads {
		device.Labels[labels.WorkloadLabel(workload.Name)] = "true"
	}
}
