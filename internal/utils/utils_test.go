package utils_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/utils"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Utils", func() {

	Context("HasFinalizer", func() {
		const finalizer = "my-finalizer"

		It("should find finalizer in CR", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}
			edgeDevice.Finalizers = append(edgeDevice.Finalizers, finalizer)

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeTrue())
		})

		It("should find finalizer in CR when among other finalizers", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}
			edgeDevice.Finalizers = append(edgeDevice.Finalizers, "other", finalizer, "and-another")

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeTrue())
		})

		It("should not find finalizer in CR when no finalizers", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeFalse())
		})

		It("should not find finalizer in CR when not among other finalizers", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}
			edgeDevice.Finalizers = append(edgeDevice.Finalizers, "other", "and-another")

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeFalse())
		})
	})

	Context("NormalizeLabel", func() {
		table.DescribeTable("should fail for an invalid format", func(tested string) {
			result, err := utils.NormalizeLabel(tested)
			Expect(result).To(Equal(""))
			Expect(err).To(HaveOccurred())
		},
			table.Entry("Empty string", ""),
			table.Entry("Non-alphanumeric characters", "$!@#$!@#$%"),
			table.Entry("Only dashes without alphanumeric characters", "-----"),
		)

		table.DescribeTable("should normalize given label to expected format", func(tested string, expected string) {
			result, err := utils.NormalizeLabel(tested)
			Expect(result).To(Equal(expected))
			Expect(err).NotTo(HaveOccurred())
		},
			table.Entry("CPU model", "Intel(R) Core(TM) i7-8665U CPU @ 1.90GHz", "intelrcoretmi7-8665ucpu1.90ghz"),
			table.Entry("CPU arch", "x86_64", "x86_64"),
			table.Entry("Serial", "PF20YKWG;", "pf20ykwg"),
		)
	})

	Context("ExtractInfoFromEnv", func() {
		It("configmaps should be empty with secret defined", func() {
			// given
			maptypes := utils.MapType{}
			envar := []v1.EnvVar{{Name: "X", ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{LocalObjectReference: v1.LocalObjectReference{Name: "secret1"}}}}}

			utils.ExtractInfoFromEnv(envar, maptypes, func(env v1.EnvVar) (bool, *bool, string, string) {
				if env.ValueFrom.ConfigMapKeyRef != nil {
					return true, env.ValueFrom.ConfigMapKeyRef.Optional, env.ValueFrom.ConfigMapKeyRef.Name, env.ValueFrom.ConfigMapKeyRef.Key
				}
				return false, nil, "", ""
			})

			// when
			_, ok := maptypes["cm1"]

			// then
			Expect(ok).To(BeFalse())
		})

		It("should extract configmaps", func() {
			// given
			maptypes := utils.MapType{}
			envar := []v1.EnvVar{{Name: "X", ValueFrom: &v1.EnvVarSource{ConfigMapKeyRef: &v1.ConfigMapKeySelector{LocalObjectReference: v1.LocalObjectReference{Name: "cm1"}}}}}

			utils.ExtractInfoFromEnv(envar, maptypes, func(env v1.EnvVar) (bool, *bool, string, string) {
				if env.ValueFrom.ConfigMapKeyRef != nil {
					return true, env.ValueFrom.ConfigMapKeyRef.Optional, env.ValueFrom.ConfigMapKeyRef.Name, env.ValueFrom.ConfigMapKeyRef.Key
				}
				return false, nil, "", ""
			})

			// when
			_, ok := maptypes["cm1"]

			// then
			Expect(ok).To(BeTrue())
		})

		It("should extract secrets", func() {
			// given
			maptypes := utils.MapType{}
			envar := []v1.EnvVar{{Name: "X", ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{LocalObjectReference: v1.LocalObjectReference{Name: "secret1"}}}}}

			utils.ExtractInfoFromEnv(envar, maptypes, func(env v1.EnvVar) (bool, *bool, string, string) {
				if env.ValueFrom.SecretKeyRef != nil {
					return true, env.ValueFrom.SecretKeyRef.Optional, env.ValueFrom.SecretKeyRef.Name, env.ValueFrom.SecretKeyRef.Key
				}
				return false, nil, "", ""
			})

			// when
			_, ok := maptypes["secret1"]

			// then
			Expect(ok).To(BeTrue())
		})
	})

	Context("ExtractInfoFromEnvFrom", func() {
		It("configmaps should be empty with secret defined", func() {
			// given
			maptypes := utils.MapType{}
			envsource := []v1.EnvFromSource{{SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: "secret1"}}}}

			utils.ExtractInfoFromEnvFrom(envsource, maptypes, func(e interface{}) (bool, *bool, string) {
				env := e.(v1.EnvFromSource)
				if env.ConfigMapRef != nil {
					return true, env.ConfigMapRef.Optional, env.ConfigMapRef.Name
				}
				return false, nil, ""
			})

			// when
			_, ok := maptypes["cm1"]

			// then
			Expect(ok).To(BeFalse())
		})

		It("should extract configmaps", func() {
			// given
			maptypes := utils.MapType{}
			envsource := []v1.EnvFromSource{{ConfigMapRef: &v1.ConfigMapEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: "cm1"}}}}

			utils.ExtractInfoFromEnvFrom(envsource, maptypes, func(e interface{}) (bool, *bool, string) {
				env := e.(v1.EnvFromSource)
				if env.ConfigMapRef != nil {
					return true, env.ConfigMapRef.Optional, env.ConfigMapRef.Name
				}
				return false, nil, ""
			})

			// when
			_, ok := maptypes["cm1"]

			// then
			Expect(ok).To(BeTrue())
		})

		It("should extract secrets", func() {
			// given
			maptypes := utils.MapType{}
			envsource := []v1.EnvFromSource{{SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: "secret1"}}}}

			utils.ExtractInfoFromEnvFrom(envsource, maptypes, func(e interface{}) (bool, *bool, string) {
				env := e.(v1.EnvFromSource)
				if env.SecretRef != nil {
					return true, env.SecretRef.Optional, env.SecretRef.Name
				}
				return false, nil, ""
			})

			// when
			_, ok := maptypes["secret1"]

			// then
			Expect(ok).To(BeTrue())
		})
	})

	Context("ExtractInfoFromVolume", func() {
		It("configmaps should be empty with secret defined", func() {
			// given
			maptypes := utils.MapType{}
			volmues := []v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: "secret1"}}}}

			utils.ExtractInfoFromVolume(volmues, maptypes, func(i interface{}) (bool, *bool, string) {
				volume := i.(v1.Volume)
				if volume.ConfigMap != nil {
					return true, volume.ConfigMap.Optional, volume.ConfigMap.Name
				}
				return false, nil, ""
			})

			// when
			_, ok := maptypes["cm1"]

			// then
			Expect(ok).To(BeFalse())
		})

		It("should extract configmaps", func() {
			// given
			maptypes := utils.MapType{}
			volmues := []v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: "cm1"}}}}}

			utils.ExtractInfoFromVolume(volmues, maptypes, func(i interface{}) (bool, *bool, string) {
				volume := i.(v1.Volume)
				if volume.ConfigMap != nil {
					return true, volume.ConfigMap.Optional, volume.ConfigMap.Name
				}
				return false, nil, ""
			})

			// when
			_, ok := maptypes["cm1"]

			// then
			Expect(ok).To(BeTrue())
		})

		It("should extract secrets", func() {
			// given
			maptypes := utils.MapType{}
			volmues := []v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: "secret1"}}}}

			utils.ExtractInfoFromVolume(volmues, maptypes, func(i interface{}) (bool, *bool, string) {
				volume := i.(v1.Volume)
				if volume.Secret != nil {
					return true, volume.Secret.Optional, volume.Secret.SecretName
				}
				return false, nil, ""
			})

			// when
			_, ok := maptypes["secret1"]

			// then
			Expect(ok).To(BeTrue())
		})
	})
})
