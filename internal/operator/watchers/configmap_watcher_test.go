package watchers_test

import (
	"time"

	"github.com/go-logr/logr"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/project-flotta/flotta-operator/internal/operator/watchers"
)

var _ = Describe("ConfigMap watcher", func() {
	var (
		configMapMock  *watchers.MockConfigMapInterface
		watcherMock    *watchers.MockInterface
		resultCh       chan watch.Event
		logger         logr.Logger
		exitCallsCount int
	)

	const (
		configMapName          = "configMap"
		configMapField         = "configMapField"
		initialFieldValue      = "value1"
		initialResourceVersion = "1"
	)

	BeforeEach(func() {
		mockCtrl := gomock.NewController(GinkgoT())
		configMapMock = watchers.NewMockConfigMapInterface(mockCtrl)
		watcherMock = watchers.NewMockInterface(mockCtrl)
		logger = zap.New()
		exitCallsCount = 0
		resultCh = make(chan watch.Event)

		configMapMock.EXPECT().Watch(gomock.Any(), gomock.Any()).Return(watcherMock, nil).AnyTimes()
		watcherMock.EXPECT().ResultChan().Return(resultCh).AnyTimes()
		watcherMock.EXPECT().Stop().AnyTimes()
	})

	Context("checkConfigMapChanges", func() {
		BeforeEach(func() {
			go func() {
				defer GinkgoRecover()
				watchers.WatchForChanges(configMapMock, configMapName, configMapField, initialFieldValue, logger, initialResourceVersion, func() { exitCallsCount++ })
			}()
		})

		AfterEach(func() {
			close(resultCh)
		})

		It("do nothing when initialFieldValue is not modified following an Add event", func() {
			// given
			configmap := corev1.ConfigMap{
				Data: map[string]string{
					configMapField: initialFieldValue,
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: initialResourceVersion,
				},
			}

			// when
			resultCh <- watch.Event{
				Type:   watch.Added,
				Object: &configmap,
			}

			// wait a bit
			<-time.After(2 * time.Second)

			// then
			Expect(exitCallsCount).To(Equal(0))
		})

		It("exit when value changed after Add event", func() {
			// given
			configmap := corev1.ConfigMap{
				Data: map[string]string{
					configMapField: "othervalue",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: initialResourceVersion,
				},
			}

			// when
			resultCh <- watch.Event{
				Type:   watch.Added,
				Object: &configmap,
			}

			// wait a bit
			<-time.After(2 * time.Second)

			// then
			Expect(exitCallsCount).To(Equal(1))
		})

		It("exit when the value changed in the configMap", func() {
			// given
			configmap := corev1.ConfigMap{
				Data: map[string]string{
					configMapField: "anothervalue",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: initialResourceVersion,
				},
			}

			// when
			resultCh <- watch.Event{
				Type:   watch.Modified,
				Object: &configmap,
			}

			// wait a bit
			<-time.After(2 * time.Second)

			// then
			Expect(exitCallsCount).To(Equal(1))
		})

		It("do nothing after Delete event", func() {
			// given
			configmap := corev1.ConfigMap{
				Data: map[string]string{
					configMapField: initialFieldValue,
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: initialResourceVersion,
				},
			}

			// when
			resultCh <- watch.Event{
				Type:   watch.Deleted,
				Object: &configmap,
			}

			// wait a bit
			<-time.After(2 * time.Second)

			// then
			Expect(exitCallsCount).To(Equal(0))
		})
	})

	Context("WatchForChanges", func() {
		It("exit from WatchForChanges if the retry wather cannot be created", func() {
			// We need to recover from panic cause the WatchForChanges will not exit (exitFunc is not os.Exit(1)) and the flow will continue but with a nil watcher which will cause
			// checkConfigMapChanges to panic.
			defer func() {
				if r := recover(); r != nil {
					// do nothing
				}
			}()

			// given
			zeroResourceVersion := "0" // will cause RetryWatcher to fail at newRetryWatcher

			// when
			watchers.WatchForChanges(configMapMock, configMapName, configMapField, initialFieldValue, logger, zeroResourceVersion, func() { exitCallsCount++ })

			// then
			Expect(exitCallsCount).To(Equal(1))
		})
	})

})
