package orgs

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/s-samadi/ghas-lab-builder/internal/config"
	api "github.com/s-samadi/ghas-lab-builder/internal/github"
	"github.com/s-samadi/ghas-lab-builder/internal/services"
	"github.com/s-samadi/ghas-lab-builder/internal/util"
	"github.com/spf13/cobra"
)

var (
	orgsFile string
)

var deleteBatchCmd = &cobra.Command{
	Use:   "delete-batch",
	Short: "Delete multiple organizations from lab environments",
	Long:  "The 'delete-batch' command lets you delete multiple organizations from GitHub Advanced Security lab environments using an organizations file.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Traverse up to find and call the root command's PersistentPreRunE
		root := cmd
		for root.Parent() != nil {
			root = root.Parent()
		}

		// Call root's PersistentPreRunE if it exists
		if root.PersistentPreRunE != nil {
			if err := root.PersistentPreRunE(cmd, args); err != nil {
				return err
			}
		}

		ctx := cmd.Context()
		cmd.SetContext(ctx)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		logger, ok := ctx.Value(config.LoggerKey).(*slog.Logger)
		if !ok || logger == nil {
			logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
		}

		startTime := time.Now()

		logger.Info("Loading organizations from file", slog.String("file", orgsFile))
		orgNames, err := util.LoadFromFile(orgsFile)
		if err != nil {
			logger.Error("Failed to load organizations file", slog.Any("error", err))
			return fmt.Errorf("failed to load organizations file: %w", err)
		}

		logger.Info("Loaded organizations", slog.Int("count", len(orgNames)))

		if len(orgNames) == 0 {
			logger.Warn("No organizations found in file")
			return nil
		}

		// Initialize delete report
		deleteReport := &services.DeleteLabReport{
			GeneratedAt:   time.Now(),
			LabDate:       "batch-delete",
			TotalUsers:    len(orgNames),
			SuccessCount:  0,
			FailureCount:  0,
			Organizations: make([]services.DeleteOrgReport, 0),
		}

		// Set up channels and workers
		orgChan := make(chan string, len(orgNames))
		resultsChan := make(chan services.DeleteOrgReport, len(orgNames))

		// Use WaitGroup to track worker goroutines
		var wg sync.WaitGroup

		// Calculate optimal number of workers: min(9, number of orgs)
		numWorkers := 9
		if len(orgNames) < numWorkers {
			numWorkers = len(orgNames)
		}

		logger.Info("Starting delete workers",
			slog.Int("worker_count", numWorkers),
			slog.Int("org_count", len(orgNames)))

		// Create worker goroutines
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerId int) {
				defer wg.Done()
				deleteOrgBatchWorker(workerId, ctx, logger, orgChan, resultsChan)
			}(i)
		}

		// Send all organizations to the channel
		for _, orgName := range orgNames {
			orgChan <- orgName
		}
		close(orgChan)

		// Close resultsChan once all workers are done
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Collect results
		resultCount := 0
		for res := range resultsChan {
			resultCount++
			deleteReport.Organizations = append(deleteReport.Organizations, res)

			if res.Status == "success" {
				deleteReport.SuccessCount++
				logger.Info("Successfully deleted organization",
					slog.String("org", res.OrgName))
			} else {
				deleteReport.FailureCount++
				logger.Error("Failed to delete organization",
					slog.String("org", res.OrgName),
					slog.String("error", res.Error))
			}
		}

		duration := time.Since(startTime)
		logger.Info("Finished batch delete",
			slog.Int("total", len(orgNames)),
			slog.Int("successful", deleteReport.SuccessCount),
			slog.Int("failed", deleteReport.FailureCount),
			slog.Duration("duration", duration))

		// Generate report
		if err := services.GenerateDeleteReportFiles(deleteReport, "reports"); err != nil {
			logger.Error("Failed to generate deletion report", slog.Any("error", err))
		} else {
			logger.Info("Generated deletion report in 'reports' directory")
		}

		if deleteReport.FailureCount > 0 {
			return fmt.Errorf("failed to delete %d organization(s)", deleteReport.FailureCount)
		}

		return nil
	},
}

// deleteOrgBatchWorker is a worker function that processes organization deletions
func deleteOrgBatchWorker(workerId int, ctx context.Context, logger *slog.Logger, orgChan chan string, resultsChan chan services.DeleteOrgReport) {
	logger.Info("Delete worker started", slog.Int("workerId", workerId))

	for orgName := range orgChan {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			logger.Warn("Delete worker stopping due to context cancellation", slog.Int("workerId", workerId))
			return
		default:
		}

		logger.Info("Deleting organization",
			slog.Int("workerId", workerId),
			slog.String("org", orgName))

		deleteTime := time.Now()
		orgReport := services.DeleteOrgReport{
			User:      "", // Not applicable for batch delete
			OrgName:   orgName,
			DeletedAt: deleteTime,
		}

		// Delete the organization
		if err := api.DeleteOrg(ctx, logger, orgName); err != nil {
			logger.Error("Failed to delete organization",
				slog.Int("workerId", workerId),
				slog.String("org", orgName),
				slog.Any("error", err))

			orgReport.Status = "failed"
			orgReport.Error = err.Error()
			resultsChan <- orgReport
			continue
		}

		orgReport.Status = "success"
		resultsChan <- orgReport
		logger.Info("Finished deleting organization",
			slog.Int("workerId", workerId),
			slog.String("org", orgName))
	}

	logger.Info("Delete worker stopped", slog.Int("workerId", workerId))
}

func init() {
	deleteBatchCmd.Flags().StringVar(&orgsFile, "orgs-file", "", "Path to organizations file (txt) containing comma-separated org names (required)")
	deleteBatchCmd.MarkFlagRequired("orgs-file")

	OrgsCmd.AddCommand(deleteBatchCmd)
}
