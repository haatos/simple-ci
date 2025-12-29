package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type pipelineSQLiteStoreSuite struct {
	pipelineStore *PipelineSQLiteStore
	db            *sql.DB
	credential    *Credential
	agent         *Agent
	suite.Suite
}

func TestPipelineSQLiteStore(t *testing.T) {
	suite.Run(t, new(pipelineSQLiteStoreSuite))
}

func (suite *pipelineSQLiteStoreSuite) SetupSuite() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	suite.db = db
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatal(err)
	}

	RunMigrations(db, "../../migrations")

	credentialStore := NewCredentialSQLiteStore(db, db)
	c, err := credentialStore.CreateCredential(
		context.Background(),
		"pipelinetestuser",
		"",
		"pipelinetestuser",
	)
	if err != nil {
		log.Fatal(err)
	}
	suite.credential = c
	agentStore := NewAgentSQLiteStore(db, db)
	a, err := agentStore.CreateAgent(
		context.Background(),
		c.CredentialID,
		"pipelineagent",
		"localhost",
		"/tmp",
		"",
		"unix",
	)
	if err != nil {
		log.Fatal(err)
	}
	suite.agent = a
	suite.pipelineStore = NewPipelineSQLiteStore(db, db)
}

func (suite *pipelineSQLiteStoreSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *pipelineSQLiteStoreSuite) TestPipelineSQLiteStore_CreatePipeline() {
	suite.Run("success - pipeline created", func() {
		// arrange
		pipelineAgentID := suite.agent.AgentID
		name := "pipeline"
		description := "description"
		repository := "git@github.com:haatos/simple-ci.git"
		scriptPath := "pipelines/testing-pipeline.yml"

		// act
		p, err := suite.pipelineStore.CreatePipeline(
			context.Background(),
			pipelineAgentID,
			name,
			description,
			repository,
			scriptPath,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(p)
		suite.NotEqual(0, p.PipelineID)
		suite.Equal(name, p.Name)
		suite.Equal(description, p.Description)
		suite.Equal(repository, p.Repository)
		suite.Equal(scriptPath, p.ScriptPath)
	})
}

func (suite *pipelineSQLiteStoreSuite) TestPipelineSQLiteStore_ReadPipelineByID() {
	suite.Run("success - pipeline found", func() {
		// arrange
		expectedPipeline := suite.createPipeline()

		// act
		p, err := suite.pipelineStore.ReadPipelineByID(
			context.Background(),
			expectedPipeline.PipelineID,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(p)
		suite.Equal(expectedPipeline.Name, p.Name)
		suite.Equal(expectedPipeline.Description, p.Description)
		suite.Equal(expectedPipeline.Repository, p.Repository)
		suite.Equal(expectedPipeline.ScriptPath, p.ScriptPath)
		suite.Equal(expectedPipeline.Schedule, p.Schedule)
	})
	suite.Run("failure - pipeline not found", func() {
		// arrange
		var id int64 = 43241

		// act
		p, err := suite.pipelineStore.ReadPipelineByID(context.Background(), id)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(p)
	})
}

func (suite *pipelineSQLiteStoreSuite) TestPipelineSQLiteStore_ReadPipelineRunData() {
	suite.Run("success - pipeline run data is found", func() {
		// arrange
		expectedPipeline := suite.createPipeline()

		// act
		prd, err := suite.pipelineStore.ReadPipelineRunData(
			context.Background(),
			expectedPipeline.PipelineID,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(prd)
		suite.Equal(expectedPipeline.PipelineID, prd.PipelineID)
		suite.Equal(expectedPipeline.Repository, prd.Repository)
		suite.Equal(expectedPipeline.ScriptPath, prd.ScriptPath)
		suite.Equal(suite.agent.AgentID, prd.AgentID)
		suite.Equal(suite.agent.Hostname, prd.Hostname)
		suite.Equal(suite.agent.Workspace, prd.Workspace)
		suite.Equal(suite.credential.CredentialID, *prd.CredentialID)
		suite.Equal(suite.credential.Username, *prd.Username)
		suite.Equal(suite.credential.SSHPrivateKeyHash, *prd.SSHPrivateKeyHash)
	})
	suite.Run("failure - pipeline run data is not found", func() {
		// arrange
		var id int64 = 42351223

		// act
		prd, err := suite.pipelineStore.ReadPipelineRunData(context.Background(), id)

		// assert
		suite.Error(err)
		suite.Nil(prd)
	})
}

func (suite *pipelineSQLiteStoreSuite) TestPipelineSQLiteStore_UpdatePipeline() {
	suite.Run("success - pipeline updates", func() {
		// arrange
		p := suite.createPipeline()
		name := "updated pipeline"
		description := "updated description"
		repository := "git@github.com:haatos/simple-cii.git"
		scriptPath := "pipelines/testing-pipeline1.yml"

		// act
		updateErr := suite.pipelineStore.UpdatePipeline(
			context.Background(),
			p.PipelineID,
			suite.agent.AgentID,
			name,
			description,
			repository,
			scriptPath,
		)
		p, readErr := suite.pipelineStore.ReadPipelineByID(
			context.Background(),
			p.PipelineID,
		)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.Equal(name, p.Name)
		suite.Equal(description, p.Description)
		suite.Equal(repository, p.Repository)
		suite.Equal(scriptPath, p.ScriptPath)
		suite.Nil(p.Schedule)
	})
}

func (suite *pipelineSQLiteStoreSuite) TestPipelineSQLiteStore_DeletePipeline() {
	suite.Run("success - pipeline is deleted", func() {
		// arrange
		p := suite.createPipeline()

		// act
		deleteErr := suite.pipelineStore.DeletePipeline(
			context.Background(),
			p.PipelineID,
		)
		p, readErr := suite.pipelineStore.ReadPipelineByID(
			context.Background(),
			p.PipelineID,
		)

		// assert
		suite.NoError(deleteErr)
		suite.Error(readErr)
		suite.Nil(p)
	})
}

func (suite *pipelineSQLiteStoreSuite) TestPipelineSQLiteStore_ListPipelines() {
	suite.Run("success - pipelines found", func() {
		// arrange
		expectedPipeline := suite.createPipeline()
		// act
		pipelines, err := suite.pipelineStore.ListPipelines(context.Background())

		// assert
		suite.NoError(err)
		suite.True(len(pipelines) >= 1)
		suite.True(slices.ContainsFunc(pipelines, func(c *Pipeline) bool {
			return c.PipelineID == expectedPipeline.PipelineID
		}))
	})
}

func (suite *pipelineSQLiteStoreSuite) TestPipelineSQLiteStore_UpdatePipelineSchedule() {
	suite.Run("success - pipeline schedule updates", func() {
		// arrange
		expectedPipeline := suite.createPipeline()

		// act
		newSchedule := "* * * * *"
		newBranch := "main"
		newJobID := uuid.NewString()

		updateErr := suite.pipelineStore.UpdatePipelineSchedule(
			context.Background(), expectedPipeline.PipelineID, &newSchedule, &newBranch, &newJobID,
		)
		p, readErr := suite.pipelineStore.ReadPipelineByID(
			context.Background(), expectedPipeline.PipelineID,
		)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.NotNil(p.Schedule)
		suite.Equal(newSchedule, *p.Schedule)
		suite.NotNil(p.ScheduleBranch)
		suite.Equal(newBranch, *p.ScheduleBranch)
		suite.NotNil(p.ScheduleJobID)
		suite.Equal(newJobID, *p.ScheduleJobID)
	})
}

func (suite *pipelineSQLiteStoreSuite) createPipeline() *Pipeline {
	p, err := suite.pipelineStore.CreatePipeline(
		context.Background(),
		suite.agent.AgentID,
		fmt.Sprintf("pipeline%d", time.Now().UTC().UnixNano()),
		fmt.Sprintf("description%d", time.Now().UTC().UnixNano()),
		"github.com:haatos/simple-ci.git",
		"/pipelines/testing.yml",
	)
	suite.NoError(err)
	return p
}
