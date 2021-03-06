package command

import (
	"context"
	"os"
	"path/filepath"

	"github.com/evergreen-ci/evergreen/model"
	"github.com/evergreen-ci/evergreen/model/task"
	"github.com/evergreen-ci/evergreen/rest/client"
	"github.com/evergreen-ci/utility"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// attachResults is used to attach MCI test results in json
// format to the task page.
type attachResults struct {
	// FileLoc describes the relative path of the file to be sent.
	// Note that this can also be described via expansions.
	FileLoc string `mapstructure:"file_location" plugin:"expand"`
	base
}

func attachResultsFactory() Command   { return &attachResults{} }
func (c *attachResults) Name() string { return "attach.results" }

// ParseParams decodes the S3 push command parameters that are
// specified as part of an AttachPlugin command; this is required
// to satisfy the 'Command' interface
func (c *attachResults) ParseParams(params map[string]interface{}) error {
	if err := mapstructure.Decode(params, c); err != nil {
		return errors.Wrapf(err, "error decoding '%v' params", c.Name())
	}

	if c.FileLoc == "" {
		return errors.New("file_location cannot be blank")
	}

	return nil
}

func (c *attachResults) expandAttachResultsParams(taskConfig *model.TaskConfig) error {
	var err error

	c.FileLoc, err = taskConfig.Expansions.ExpandString(c.FileLoc)
	if err != nil {
		return errors.Wrap(err, "error expanding file_location")
	}

	return nil
}

// Execute carries out the attachResults command - this is required
// to satisfy the 'Command' interface
func (c *attachResults) Execute(ctx context.Context,
	comm client.Communicator, logger client.LoggerProducer, conf *model.TaskConfig) error {

	if err := c.expandAttachResultsParams(conf); err != nil {
		return errors.WithStack(err)
	}

	reportFileLoc := c.FileLoc
	if !filepath.IsAbs(c.FileLoc) {
		reportFileLoc = filepath.Join(conf.WorkDir, c.FileLoc)
	}

	// attempt to open the file
	reportFile, err := os.Open(reportFileLoc)
	if err != nil {
		return errors.Wrapf(err, "Couldn't open report file '%s'", reportFileLoc)
	}
	defer reportFile.Close()

	results := &task.LocalTestResults{}
	if err = utility.ReadJSON(reportFile, results); err != nil {
		return errors.Wrapf(err, "Couldn't read report file '%s'", reportFileLoc)
	}

	return errors.WithStack(sendJSONResults(ctx, conf, logger, comm, results))
}
