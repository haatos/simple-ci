package store

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal"
	"github.com/stretchr/testify/assert"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestRunSQLiteStore_CreateRun(t *testing.T) {
	t.Run("success - run created", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)
		branch := "main"

		// act
		r, err := runStore.CreateRun(
			context.Background(),
			p.PipelineID,
			branch,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, branch, r.Branch)
		assert.Equal(t, StatusQueued, r.Status)
	})
	t.Run("failure - invalid pipeline id", func(t *testing.T) {
		// arrange
		var pipelineID int64 = 2345523

		// act
		p, err := runStore.CreateRun(context.Background(), pipelineID, "main")

		// assert
		assert.Error(t, err)
		var sqliteErr *sqlite.Error
		ok := errors.As(err, &sqliteErr)
		assert.True(t, ok)
		assert.Equal(t, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY, sqliteErr.Code())
		assert.Nil(t, p)
	})
}

func TestRunSQLiteStore_ReadRunByID(t *testing.T) {
	t.Run("success - run is found", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)
		expectedRun, err := runStore.CreateRun(
			context.Background(), p.PipelineID, "main")
		assert.NoError(t, err)

		// act
		r, err := runStore.ReadRunByID(context.Background(), expectedRun.RunID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, expectedRun.Branch, r.Branch)
		assert.Equal(t, expectedRun.Status, r.Status)
	})
	t.Run("failure - run is not found", func(t *testing.T) {
		// arrange
		var runID int64 = 3432452

		// act
		r, err := runStore.ReadRunByID(context.Background(), runID)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, r)
	})
}

func TestRunSQLiteStore_UpdateRunStartedOn(t *testing.T) {
	t.Run("success - run started on updates", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)
		expectedRun, err := runStore.CreateRun(
			context.Background(), p.PipelineID, "main")
		assert.NoError(t, err)

		// act
		now := time.Now().UTC()
		updateErr := runStore.UpdateRunStartedOn(
			context.Background(),
			expectedRun.RunID,
			time.Now().UTC().Format(internal.RunDirLayout),
			StatusRunning,
			&now,
		)
		r, readErr := runStore.ReadRunByID(
			context.Background(), expectedRun.RunID)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.NotNil(t, r)
		assert.Equal(
			t,
			now.Format(internal.DBTimestampLayout),
			r.StartedOn.Format(internal.DBTimestampLayout),
		)
	})
}

func TestRunSQLiteStore_UpdateRunEndedOn(t *testing.T) {
	t.Run("success - run ended on updates", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)
		expectedRun, err := runStore.CreateRun(
			context.Background(), p.PipelineID, "main")
		assert.NoError(t, err)

		// act
		output := "test output"
		artifacts := "artifacts.zip"
		now := time.Now().UTC()
		updateErr := runStore.UpdateRunEndedOn(
			context.Background(),
			expectedRun.RunID,
			StatusPassed,
			&output,
			&artifacts,
			&now,
		)
		r, readErr := runStore.ReadRunByID(
			context.Background(), expectedRun.RunID)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.NotNil(t, r)
		assert.Equal(
			t,
			now.Format(internal.DBTimestampLayout),
			r.EndedOn.Format(internal.DBTimestampLayout),
		)
		assert.Equal(t, output, *r.Output)
		assert.Equal(t, artifacts, *r.Artifacts)
	})
}

func TestRunSQLiteStore_DeleteRun(t *testing.T) {
	t.Run("success - run is deleted", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)
		expectedRun, err := runStore.CreateRun(
			context.Background(), p.PipelineID, "main")
		assert.NoError(t, err)

		// act
		deleteErr := runStore.DeleteRun(
			context.Background(), expectedRun.RunID)
		r, readErr := runStore.ReadRunByID(
			context.Background(), expectedRun.RunID)

		// assert
		assert.NoError(t, deleteErr)
		assert.Error(t, readErr)
		assert.True(t, errors.Is(readErr, sql.ErrNoRows))
		assert.Nil(t, r)
	})
}

func TestRunSQLiteStore_ListPipelineRuns(t *testing.T) {
	t.Run("success - pipeline runs found", func(t *testing.T) {
		// arrange
		c := createCredential(t)
		a := createAgent(t, c)
		p := createPipeline(t, a)
		expectedRun, err := runStore.CreateRun(
			context.Background(), p.PipelineID, "main")
		assert.NoError(t, err)

		// act
		runs, err := runStore.ListPipelineRuns(
			context.Background(), p.PipelineID)

		// assert
		assert.NoError(t, err)
		assert.True(t, len(runs) >= 1)
		assert.True(t, slices.ContainsFunc(runs, func(r Run) bool {
			return expectedRun.RunID == r.RunID
		}))
	})
}
