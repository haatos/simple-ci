package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPipelineSQLiteStore_CreatePipeline(t *testing.T) {
	t.Run("success - pipeline created", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		pipelineAgentID := a.AgentID
		name := "pipeline"
		description := "description"
		repository := "git@github.com:haatos/simple-ci.git"
		scriptPath := "pipelines/testing-pipeline.yml"

		// act
		p, err := pipelineStore.CreatePipeline(
			context.Background(),
			pipelineAgentID,
			name,
			description,
			repository,
			scriptPath,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.NotEqual(t, 0, p.PipelineID)
		assert.Equal(t, name, p.Name)
		assert.Equal(t, description, p.Description)
		assert.Equal(t, repository, p.Repository)
		assert.Equal(t, scriptPath, p.ScriptPath)
	})
}

func TestPipelineSQLiteStore_ReadPipelineByID(t *testing.T) {
	t.Run("success - pipeline found", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		expectedPipeline := createPipeline(t, a)

		// act
		p, err := pipelineStore.ReadPipelineByID(
			context.Background(),
			expectedPipeline.PipelineID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.Equal(t, expectedPipeline.Name, p.Name)
		assert.Equal(t, expectedPipeline.Description, p.Description)
		assert.Equal(t, expectedPipeline.Repository, p.Repository)
		assert.Equal(t, expectedPipeline.ScriptPath, p.ScriptPath)
		assert.Equal(t, expectedPipeline.Schedule, p.Schedule)
	})
	t.Run("failure - pipeline not found", func(t *testing.T) {
		// arrange
		var id int64 = 43241

		// act
		p, err := pipelineStore.ReadPipelineByID(context.Background(), id)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, p)
	})
}

func TestPipelineSQLiteStore_UpdatePipeline(t *testing.T) {
	t.Run("success - pipeline updates", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)
		name := "updated pipeline"
		description := "updated description"
		repository := "git@github.com:haatos/simple-cii.git"
		scriptPath := "pipelines/testing-pipeline1.yml"

		// act
		updateErr := pipelineStore.UpdatePipeline(
			context.Background(),
			p.PipelineID,
			a.AgentID,
			name,
			description,
			repository,
			scriptPath,
		)
		p, readErr := pipelineStore.ReadPipelineByID(
			context.Background(),
			p.PipelineID,
		)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.Equal(t, name, p.Name)
		assert.Equal(t, description, p.Description)
		assert.Equal(t, repository, p.Repository)
		assert.Equal(t, scriptPath, p.ScriptPath)
		assert.Nil(t, p.Schedule)
	})
}

func TestPipelineSQLiteStore_DeletePipeline(t *testing.T) {
	t.Run("success - pipeline is deleted", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)

		// act
		deleteErr := pipelineStore.DeletePipeline(
			context.Background(),
			p.PipelineID,
		)
		p, readErr := pipelineStore.ReadPipelineByID(
			context.Background(),
			p.PipelineID,
		)

		// assert
		assert.NoError(t, deleteErr)
		assert.Error(t, readErr)
		assert.Nil(t, p)
	})
}

func TestPipelineSQLiteStore_ListPipelines(t *testing.T) {
	t.Run("success - pipelines found", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		expectedPipeline := createPipeline(t, a)
		// act
		pipelines, err := pipelineStore.ListPipelines(context.Background())

		// assert
		assert.NoError(t, err)
		assert.True(t, len(pipelines) >= 1)
		assert.True(t, slices.ContainsFunc(pipelines, func(c *Pipeline) bool {
			return c.PipelineID == expectedPipeline.PipelineID
		}))
	})
}

func TestPipelineSQLiteStore_UpdatePipelineSchedule(t *testing.T) {
	t.Run("success - pipeline schedule updates", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		expectedPipeline := createPipeline(t, a)

		// act
		newSchedule := "* * * * *"
		newBranch := "main"
		newJobID := uuid.NewString()

		updateErr := pipelineStore.UpdatePipelineSchedule(
			context.Background(), expectedPipeline.PipelineID, &newSchedule, &newBranch, &newJobID,
		)
		p, readErr := pipelineStore.ReadPipelineByID(
			context.Background(), expectedPipeline.PipelineID,
		)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.NotNil(t, p.Schedule)
		assert.Equal(t, newSchedule, *p.Schedule)
		assert.NotNil(t, p.ScheduleBranch)
		assert.Equal(t, newBranch, *p.ScheduleBranch)
		assert.NotNil(t, p.ScheduleJobID)
		assert.Equal(t, newJobID, *p.ScheduleJobID)
	})
}

func createPipeline(t *testing.T, a *Agent) *Pipeline {
	p, err := pipelineStore.CreatePipeline(
		context.Background(),
		a.AgentID,
		fmt.Sprintf("pipeline%d", time.Now().UnixNano()),
		fmt.Sprintf("description%d", time.Now().UnixNano()),
		"github.com:haatos/simple-ci.git",
		"/pipelines/testing.yml",
	)
	assert.NoError(t, err)
	return p
}
