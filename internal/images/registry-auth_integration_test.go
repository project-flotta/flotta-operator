package images_test

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/project-flotta/flotta-operator/internal/images"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testAuthFile = `{
    "auths": {
        "https://index.docker.io/v1/": {
            "auth": "dGVzdC10ZXN0Cg=="
        }
    }
}`
)

var _ = Describe("Images", func() {
	var (
		registryAuth *images.RegistryAuth
	)

	BeforeEach(func() {
		registryAuth = images.NewRegistryAuth(k8sClient)
	})

	It("should get authfile from a secret", func() {
		// given
		data := map[string][]byte{
			".dockerconfigjson": []byte(testAuthFile),
		}
		createSecret("default", "test-auth", data)

		// when
		authFile, err := registryAuth.GetAuthFileFromSecret(context.TODO(), "default", "test-auth")

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(authFile).To(Equal(testAuthFile))
	})

	It("should report error when secret is not found", func() {
		// when
		_, err := registryAuth.GetAuthFileFromSecret(context.TODO(), "default", "test-auth-missing")

		// then
		Expect(err).To(HaveOccurred())
	})

	It("should report error when .dockerconfigjson key is not found", func() {
		// given
		secretName := "test-auth-no-dockerconfigjson"
		data := map[string][]byte{
			"authfile": []byte(testAuthFile),
		}
		createSecret("default", secretName, data)

		// when
		_, err := registryAuth.GetAuthFileFromSecret(context.TODO(), "default", secretName)

		// then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(".dockerconfigjson"))
	})
})

func createSecret(namespace, name string, data map[string][]byte) {
	secret := corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       data,
	}
	err := k8sClient.Create(context.TODO(), &secret)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}
