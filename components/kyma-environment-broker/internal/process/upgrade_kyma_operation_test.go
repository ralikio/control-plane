package process

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeKymaOperationManager_OperationSucceeded(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewUpgradeKymaOperationManager(operations)
	op := fixUpgradeKymaOperation()
	err := operations.InsertUpgradeKymaOperation(op)
	require.NoError(t, err)

	// when
	op, when, err := opManager.OperationSucceeded(op, "task succeeded")

	// then
	assert.NoError(t, err)
	assert.Equal(t, domain.Succeeded, op.State)
	assert.Equal(t, time.Duration(0), when)
}

func TestUpgradeKymaOperationManager_OperationFailed(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewUpgradeKymaOperationManager(operations)
	op := fixUpgradeKymaOperation()
	err := operations.InsertUpgradeKymaOperation(op)
	require.NoError(t, err)

	errMsg := "task failed miserably"

	// when
	op, when, err := opManager.OperationFailed(op, errMsg)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, errMsg)
	assert.Equal(t, domain.Failed, op.State)
	assert.Equal(t, time.Duration(0), when)
}

func TestUpgradeKymaOperationManager_RetryOperation(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewUpgradeKymaOperationManager(operations)
	op := internal.UpgradeKymaOperation{}
	op.UpdatedAt = time.Now()
	retryInterval := time.Hour
	errorMessage := fmt.Sprintf("task failed")
	maxtime := time.Hour * 3 // allow 2 retries

	// this is required to avoid storage retries (without this statement there will be an error => retry)
	err := operations.InsertUpgradeKymaOperation(op)
	require.NoError(t, err)

	// then - first call
	op, when, err := opManager.RetryOperation(op, errorMessage, retryInterval, maxtime, fixLogger())

	// when - first retry
	assert.True(t, when > 0)
	assert.Nil(t, err)

	// then - second call
	t.Log(op.UpdatedAt.String())
	op.UpdatedAt = op.UpdatedAt.Add(-retryInterval - time.Second) // simulate wait of first retry
	t.Log(op.UpdatedAt.String())
	op, when, err = opManager.RetryOperation(op, errorMessage, retryInterval, maxtime, fixLogger())

	// when - second call => retry
	assert.True(t, when > 0)
	assert.Nil(t, err)

}

func fixUpgradeKymaOperation() internal.UpgradeKymaOperation {
	return internal.UpgradeKymaOperation{
		Operation: internal.Operation{
			ID:                     "2c538027-d1c4-41ef-a26c-c9604483cb6d",
			Version:                0,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Time{},
			InstanceID:             "2b6645a1-87e7-491d-bce3-cc0fbe16b6c0",
			ProvisionerOperationID: "",
			State:                  domain.InProgress,
			Description:            "op description",
			ProvisioningParameters: internal.ProvisioningParameters{},
		},
		RuntimeOperation: orchestration.RuntimeOperation{
			Runtime: orchestration.Runtime{
				SubAccountID: "",
				RuntimeID:    "",
			},
			DryRun: false,
		},
		InputCreator: nil,
	}
}
