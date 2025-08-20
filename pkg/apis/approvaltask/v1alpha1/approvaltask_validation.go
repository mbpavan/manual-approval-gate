/*
Copyright 2022 The OpenShift Pipelines Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*ApprovalTask)(nil)

// Validate ApprovalTask
func (tg *ApprovalTask) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(tg.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return tg.Spec.Validate(ctx)
}

// Validate ApprovalTaskSpec
func (tgs *ApprovalTaskSpec) Validate(ctx context.Context) (errs *apis.FieldError) {
	if tgs.NumberOfApprovalsRequired <= 0 {
		errs = errs.Also(apis.ErrInvalidValue(tgs.NumberOfApprovalsRequired, "numberOfApprovalsRequired", "must be greater than 0"))
	}

	if len(tgs.Approvers) == 0 {
		errs = errs.Also(apis.ErrMissingField("approvers"))
	}

	// Validate approvers
	for i, approver := range tgs.Approvers {
		fieldPath := fmt.Sprintf("approvers[%d]", i)
		
		if err := validateApprover(approver, fieldPath); err != nil {
			errs = errs.Also(err)
		}
	}

	// Note: We don't validate numberOfApprovalsRequired vs approver count because:
	// 1. Groups can have multiple members (unknown at validation time)  
	// 2. Group membership is resolved at runtime, not validation time
	// 3. If there aren't enough approvers at runtime, the task stays "pending" (correct behavior)

	return errs
}

// validateApprover validates a single approver entry
func validateApprover(approver ApproverDetails, fieldPath string) *apis.FieldError {
	var errs *apis.FieldError

	// Validate approver name
	if strings.TrimSpace(approver.Name) == "" {
		errs = errs.Also(apis.ErrMissingField(fieldPath + ".name"))
		return errs
	}

	// Validate approver type
	approverType := DefaultedApproverType(approver.Type)
	if approverType != "User" && approverType != "Group" {
		errs = errs.Also(apis.ErrInvalidValue(approver.Type, fieldPath+".type", "must be either 'User' or 'Group'"))
	}

	// Validate name format based on type
	if approverType == "User" {
		if err := validateUserName(approver.Name); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(approver.Name, fieldPath+".name", err.Error()))
		}
	} else if approverType == "Group" {
		if err := validateGroupName(approver.Name); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(approver.Name, fieldPath+".name", err.Error()))
		}
	}

	// Validate input value
	validInputs := []string{"pending", "approve", "reject"}
	if !contains(validInputs, approver.Input) {
		errs = errs.Also(apis.ErrInvalidValue(approver.Input, fieldPath+".input", 
			fmt.Sprintf("must be one of: %s", strings.Join(validInputs, ", "))))
	}

	// Validate users for group type
	if approverType == "Group" {
		for j, user := range approver.Users {
			userFieldPath := fmt.Sprintf("%s.users[%d]", fieldPath, j)
			
			if strings.TrimSpace(user.Name) == "" {
				errs = errs.Also(apis.ErrMissingField(userFieldPath + ".name"))
			} else if err := validateUserName(user.Name); err != nil {
				errs = errs.Also(apis.ErrInvalidValue(user.Name, userFieldPath+".name", err.Error()))
			}

			if !contains(validInputs, user.Input) {
				errs = errs.Also(apis.ErrInvalidValue(user.Input, userFieldPath+".input", 
					fmt.Sprintf("must be one of: %s", strings.Join(validInputs, ", "))))
			}
		}
	}

	return errs
}

// validateUserName validates username format.
func validateUserName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("username cannot be empty")
	}
	
	if strings.Contains(name, ":") {
		return fmt.Errorf("username cannot contain colons - use 'group:groupname' format for groups")
	}
	
	return nil
}

// validateGroupName validates group name format.
func validateGroupName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	
	if strings.Contains(name, ":") {
		return fmt.Errorf("group name cannot contain colons")
	}
	
	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
