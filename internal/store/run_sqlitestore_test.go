package store

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"slices"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal"
	"github.com/stretchr/testify/suite"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type runSQLiteStoreSuite struct {
	runStore   *RunSQLiteStore
	db         *sql.DB
	credential *Credential
	agent      *Agent
	pipeline   *Pipeline
	suite.Suite
}

func TestRunSQLiteStore(t *testing.T) {
	suite.Run(t, new(runSQLiteStoreSuite))
}

func (suite *runSQLiteStoreSuite) SetupSuite() {
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

	suite.runStore = NewRunSQLiteStore(db, db)
	credentialStore := NewCredentialSQLiteStore(db, db)
	c, err := credentialStore.CreateCredential(
		context.Background(),
		"runtestuser",
		"",
		"runtestuser",
	)
	if err != nil {
		log.Fatal(err)
	}
	suite.credential = c
	agentStore := NewAgentSQLiteStore(db, db)
	a, err := agentStore.CreateAgent(
		context.Background(),
		c.CredentialID,
		"runagent",
		"localhost",
		"/tmp",
		"",
		"unix",
	)
	if err != nil {
		log.Fatal(err)
	}
	suite.agent = a
	pipelineStore := NewPipelineSQLiteStore(db, db)
	p, err := pipelineStore.CreatePipeline(
		context.Background(),
		a.AgentID,
		"runpipeline",
		"",
		"github.com:haatos/simple-ci.git",
		"pipelines/testing-pipeline.yml",
	)
	if err != nil {
		log.Fatal(err)
	}
	suite.pipeline = p
}

func (suite *runSQLiteStoreSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *runSQLiteStoreSuite) TestRunSQLiteStore_CreatePipelineRun() {
	suite.Run("success - run created", func() {
		// arrange
		branch := "main"

		// act
		r, err := suite.runStore.CreatePipelineRun(
			context.Background(),
			suite.pipeline.PipelineID,
			branch,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(r)
		suite.Equal(branch, r.Branch)
		suite.Equal(StatusQueued, r.Status)
	})
	suite.Run("failure - invalid pipeline id", func() {
		// arrange
		var pipelineID int64 = 2345523

		// act
		p, err := suite.runStore.CreatePipelineRun(context.Background(), pipelineID, "main")

		// assert
		suite.Error(err)
		var sqliteErr *sqlite.Error
		ok := errors.As(err, &sqliteErr)
		suite.True(ok)
		suite.Equal(sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY, sqliteErr.Code())
		suite.Nil(p)
	})
}

func (suite *runSQLiteStoreSuite) TestRunSQLiteStore_ReadRunByID() {
	suite.Run("success - run is found", func() {
		// arrange
		expectedRun, err := suite.runStore.CreatePipelineRun(
			context.Background(), suite.pipeline.PipelineID, "main")
		suite.NoError(err)

		// act
		r, err := suite.runStore.ReadRunByID(context.Background(), expectedRun.RunID)

		// assert
		suite.NoError(err)
		suite.NotNil(r)
		suite.Equal(expectedRun.Branch, r.Branch)
		suite.Equal(expectedRun.Status, r.Status)
	})
	suite.Run("failure - run is not found", func() {
		// arrange
		var runID int64 = 3432452

		// act
		r, err := suite.runStore.ReadRunByID(context.Background(), runID)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(r)
	})
}

func (suite *runSQLiteStoreSuite) TestRunSQLiteStore_UpdatePipelineRunStartedOn() {
	suite.Run("success - run started on updates", func() {
		// arrange
		expectedRun, err := suite.runStore.CreatePipelineRun(
			context.Background(), suite.pipeline.PipelineID, "main")
		suite.NoError(err)

		// act
		now := time.Now().UTC()
		updateErr := suite.runStore.UpdatePipelineRunStartedOn(
			context.Background(),
			expectedRun.RunID,
			time.Now().UTC().Format(internal.RunDirLayout),
			StatusRunning,
			&now,
		)
		r, readErr := suite.runStore.ReadRunByID(
			context.Background(), expectedRun.RunID)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.NotNil(r)
		suite.Equal(&now, r.StartedOn)
	})
}

func (suite *runSQLiteStoreSuite) TestRunSQLiteStore_UpdatePipelineRunEndedOn() {
	suite.Run("success - run ended on updates", func() {
		// arrange
		expectedRun, err := suite.runStore.CreatePipelineRun(
			context.Background(), suite.pipeline.PipelineID, "main")
		suite.NoError(err)

		// act
		artifacts := "artifacts.zip"
		now := time.Now().UTC()
		updateErr := suite.runStore.UpdatePipelineRunEndedOn(
			context.Background(),
			expectedRun.RunID,
			StatusPassed,
			&artifacts,
			&now,
		)
		r, readErr := suite.runStore.ReadRunByID(
			context.Background(), expectedRun.RunID)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.NotNil(r)
		suite.Equal(&now, r.EndedOn)
		suite.Equal(artifacts, *r.Artifacts)
	})
}

func (suite *runSQLiteStoreSuite) TestRunSQLiteStore_DeletePipelineRun() {
	suite.Run("success - run is deleted", func() {
		// arrange
		expectedRun, err := suite.runStore.CreatePipelineRun(
			context.Background(), suite.pipeline.PipelineID, "main")
		suite.NoError(err)

		// act
		deleteErr := suite.runStore.DeletePipelineRun(
			context.Background(), expectedRun.RunID)
		r, readErr := suite.runStore.ReadRunByID(
			context.Background(), expectedRun.RunID)

		// assert
		suite.NoError(deleteErr)
		suite.Error(readErr)
		suite.True(errors.Is(readErr, sql.ErrNoRows))
		suite.Nil(r)
	})
}

func (suite *runSQLiteStoreSuite) TestRunSQLiteStore_ListPipelineRuns() {
	suite.Run("success - pipeline runs found", func() {
		// arrange
		expectedRun, err := suite.runStore.CreatePipelineRun(
			context.Background(), suite.pipeline.PipelineID, "main")
		suite.NoError(err)

		// act
		runs, err := suite.runStore.ListPipelineRuns(
			context.Background(), suite.pipeline.PipelineID)

		// assert
		suite.NoError(err)
		suite.True(len(runs) >= 1)
		suite.True(slices.ContainsFunc(runs, func(r Run) bool {
			return expectedRun.RunID == r.RunID
		}))
	})
}
