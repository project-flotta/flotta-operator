package mtls

import "fmt"

type NoClientCertSendError struct{}

func (e *NoClientCertSendError) Error() string {
	return "no Client cert was sent"
}

type RegisterClientVerifyError struct {
	err error
}

func (e *RegisterClientVerifyError) Error() string {
	return fmt.Sprintf("cannot verify register certificate: %v", e.err)
}

type InvalidCertificateKindError struct{}

func (e *InvalidCertificateKindError) Error() string {
	return "cannot use register certificate on this resource"
}

type ClientCertificateVerifyError struct {
	err error
}

func (e *ClientCertificateVerifyError) Error() string {
	return fmt.Sprintf("cannot verify certificate: %v", e.err)
}
