// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// PlaybookExecution playbook execution
//
// swagger:model playbook-execution
type PlaybookExecution struct {

	// Returns the ansible playbook as a string.
	AnsiblePlaybookString string `json:"ansible-playbook-string,omitempty"`

	// last data upload
	// Format: date-time
	LastDataUpload strfmt.DateTime `json:"last_data_upload,omitempty"`

	// Returns the ansible playbookexecution name.
	Name string `json:"name,omitempty"`
}

// Validate validates this playbook execution
func (m *PlaybookExecution) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateLastDataUpload(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *PlaybookExecution) validateLastDataUpload(formats strfmt.Registry) error {
	if swag.IsZero(m.LastDataUpload) { // not required
		return nil
	}

	if err := validate.FormatOf("last_data_upload", "body", "date-time", m.LastDataUpload.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this playbook execution based on context it is used
func (m *PlaybookExecution) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *PlaybookExecution) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PlaybookExecution) UnmarshalBinary(b []byte) error {
	var res PlaybookExecution
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
