//
package pipeline

import (
	"github.com/getgauge-contrib/gauge-go/gauge"
	m "github.com/getgauge-contrib/gauge-go/models"
	"github.com/openshift-pipelines/release-tests/pkg/pipelines"
	"github.com/openshift-pipelines/release-tests/pkg/store"
)

var _ = gauge.Step("Verify taskrun <table>", func(table *m.Table) {
	for _, row := range table.Rows {
		trname := row.Cells[1]
		status := row.Cells[2]
		pipelines.ValidateTaskRun(store.Clients(), trname, status, store.Namespace())
	}
})

var _ = gauge.Step("Verify pipelinerun <table>", func(table *m.Table) {
	for _, row := range table.Rows {
		prname := row.Cells[1]
		status := row.Cells[2]
		labelCheck := row.Cells[3]
		pipelines.ValidatePipelineRun(store.Clients(), prname, status, labelCheck, store.Namespace())
	}
})

var _ = gauge.Step("Watch for pipelinerun resources", func() {
	pipelines.WatchForPipelineRun(store.Clients(), store.Namespace())
})

var _ = gauge.Step("Verify taskrun <trname> label propagation", func(trname string) {
	pipelines.ValidateTaskRunLabelPropogation(store.Clients(), trname, store.Namespace())
})

var _ = gauge.Step("Assert no new pipelineruns created", func() {
	pipelines.AssertForNoNewPipelineRunCreation(store.Clients(), store.Namespace())
})

var _ = gauge.Step("<numberOfPr> pipelinerun(s) should be present within <timeoutSeconds> seconds", func(numberOfPr, timeoutSeconds string) {
	pipelines.AssertNumberOfPipelineruns(store.Clients(), store.Namespace(), numberOfPr, timeoutSeconds)
})

var _ = gauge.Step("<numberOfTr> taskrun(s) should be present within <timeoutSeconds> seconds", func(numberOfTr, timeoutSeconds string) {
	pipelines.AssertNumberOfTaskruns(store.Clients(), store.Namespace(), numberOfTr, timeoutSeconds)
})
