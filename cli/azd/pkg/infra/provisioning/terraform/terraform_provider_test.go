package terraform

import (
	"context"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/environment"
	"github.com/azure/azure-dev/cli/azd/pkg/executil"
	"github.com/azure/azure-dev/cli/azd/pkg/infra"
	. "github.com/azure/azure-dev/cli/azd/pkg/infra/provisioning"
	"github.com/azure/azure-dev/cli/azd/test/mocks"
	execmock "github.com/azure/azure-dev/cli/azd/test/mocks/executil"
	"github.com/stretchr/testify/require"
)

func TestTerraformPlan(t *testing.T) {
	progressLog := []string{}
	interactiveLog := []bool{}
	progressDone := make(chan bool)

	mockContext := mocks.NewMockContext(context.Background())
	prepareGenericMocks(mockContext.CommandRunner)
	preparePlanningMocks(mockContext.CommandRunner)

	infraProvider := createTerraformProvider(*mockContext.Context)
	planningTask := infraProvider.Plan(*mockContext.Context)

	go func() {
		for progressReport := range planningTask.Progress() {
			progressLog = append(progressLog, progressReport.Message)
		}
		progressDone <- true
	}()

	go func() {
		for planningInteractive := range planningTask.Interactive() {
			interactiveLog = append(interactiveLog, planningInteractive)
		}
	}()

	deploymentPlan, err := planningTask.Await()
	<-progressDone

	require.Nil(t, err)
	require.NotNil(t, deploymentPlan.Deployment)

	require.Len(t, progressLog, 7)
	require.Contains(t, progressLog[0], "Initialize terraform")
	require.Contains(t, progressLog[1], "Generating terraform parameters")
	require.Contains(t, progressLog[2], "Validate terraform template")
	require.Contains(t, progressLog[3], "terraform validate result : Success! The configuration is valid.")
	require.Contains(t, progressLog[4], "Plan terraform template")
	require.Contains(t, progressLog[5], "terraform plan result : To perform exactly these actions, run the following command to apply:terraform apply")
	require.Contains(t, progressLog[6], "Create terraform template")

	require.Equal(t, infraProvider.env.Values["AZURE_LOCATION"], deploymentPlan.Deployment.Parameters["location"].Value)
	require.Equal(t, infraProvider.env.Values["AZURE_ENV_NAME"], deploymentPlan.Deployment.Parameters["name"].Value)

	require.NotNil(t, deploymentPlan.Details)

	terraformDeploymentData := deploymentPlan.Details.(TerraformDeploymentDetails)
	require.NotNil(t, terraformDeploymentData)

	require.FileExists(t, terraformDeploymentData.ParameterFilePath)
	require.NotEmpty(t, terraformDeploymentData.ParameterFilePath)
	require.NotEmpty(t, terraformDeploymentData.localStateFilePath)
}

func TestTerraformDeploy(t *testing.T) {
	progressLog := []string{}
	interactiveLog := []bool{}
	progressDone := make(chan bool)

	mockContext := mocks.NewMockContext(context.Background())
	prepareGenericMocks(mockContext.CommandRunner)
	preparePlanningMocks(mockContext.CommandRunner)
	prepareDeployMocks(mockContext.CommandRunner)

	infraProvider := createTerraformProvider(*mockContext.Context)

	envPath := path.Join(infraProvider.projectPath, ".azure", infraProvider.env.Values["AZURE_ENV_NAME"])

	deploymentPlan := DeploymentPlan{
		Details: TerraformDeploymentDetails{
			ParameterFilePath:  path.Join(envPath, "main.tfvars.json"),
			PlanFilePath:       path.Join(envPath, "main.tfplan"),
			localStateFilePath: path.Join(envPath, "terraform.tfstate"),
		},
	}

	scope := infra.NewSubscriptionScope(*mockContext.Context, infraProvider.env.Values["AZURE_LOCATION"], infraProvider.env.GetSubscriptionId(), infraProvider.env.GetEnvName())
	deployTask := infraProvider.Deploy(*mockContext.Context, &deploymentPlan, scope)

	go func() {
		for deployProgress := range deployTask.Progress() {
			progressLog = append(progressLog, deployProgress.Message)
		}
		progressDone <- true
	}()

	go func() {
		for deployInteractive := range deployTask.Interactive() {
			interactiveLog = append(interactiveLog, deployInteractive)
		}
	}()

	deployResult, err := deployTask.Await()
	<-progressDone

	require.Nil(t, err)
	require.NotNil(t, deployResult)

	require.Equal(t, deployResult.Deployment.Outputs["AZURE_LOCATION"].Value, infraProvider.env.Values["AZURE_LOCATION"])
	require.Equal(t, deployResult.Deployment.Outputs["RG_NAME"].Value, fmt.Sprintf("rg-%s", infraProvider.env.GetEnvName()))
}

func TestTerraformDestroy(t *testing.T) {
	mockContext := mocks.NewMockContext(context.Background())
	prepareGenericMocks(mockContext.CommandRunner)
	preparePlanningMocks(mockContext.CommandRunner)
	prepareDestroyMocks(mockContext.CommandRunner)

	progressLog := []string{}
	interactiveLog := []bool{}
	progressDone := make(chan bool)

	infraProvider := createTerraformProvider(*mockContext.Context)
	deployment := Deployment{}

	destroyOptions := NewDestroyOptions(false, false)
	destroyTask := infraProvider.Destroy(*mockContext.Context, &deployment, destroyOptions)

	go func() {
		for destroyProgress := range destroyTask.Progress() {
			progressLog = append(progressLog, destroyProgress.Message)
		}
		progressDone <- true
	}()

	go func() {
		for destroyInteractive := range destroyTask.Interactive() {
			interactiveLog = append(interactiveLog, destroyInteractive)
		}
	}()

	destroyResult, err := destroyTask.Await()
	<-progressDone

	require.Nil(t, err)
	require.NotNil(t, destroyResult)

	require.Equal(t, destroyResult.Outputs["AZURE_LOCATION"].Value, infraProvider.env.Values["AZURE_LOCATION"])
	require.Equal(t, destroyResult.Outputs["RG_NAME"].Value, fmt.Sprintf("rg-%s", infraProvider.env.GetEnvName()))
}

func TestTerraformGetDeployment(t *testing.T) {
	progressLog := []string{}
	interactiveLog := []bool{}
	progressDone := make(chan bool)

	mockContext := mocks.NewMockContext(context.Background())
	prepareGenericMocks(mockContext.CommandRunner)
	preparePlanningMocks(mockContext.CommandRunner)
	prepareDeployMocks(mockContext.CommandRunner)

	infraProvider := createTerraformProvider(*mockContext.Context)
	scope := infra.NewSubscriptionScope(*mockContext.Context, infraProvider.env.Values["AZURE_LOCATION"], infraProvider.env.GetSubscriptionId(), infraProvider.env.GetEnvName())
	getDeploymentTask := infraProvider.GetDeployment(*mockContext.Context, scope)

	go func() {
		for progressReport := range getDeploymentTask.Progress() {
			progressLog = append(progressLog, progressReport.Message)
		}
		progressDone <- true
	}()

	go func() {
		for deploymentInteractive := range getDeploymentTask.Interactive() {
			interactiveLog = append(interactiveLog, deploymentInteractive)
		}
	}()

	getDeploymentResult, err := getDeploymentTask.Await()
	<-progressDone

	require.Nil(t, err)
	require.NotNil(t, getDeploymentResult.Deployment)

	require.Equal(t, getDeploymentResult.Deployment.Outputs["AZURE_LOCATION"].Value, infraProvider.env.Values["AZURE_LOCATION"])
	require.Equal(t, getDeploymentResult.Deployment.Outputs["RG_NAME"].Value, fmt.Sprintf("rg-%s", infraProvider.env.GetEnvName()))

}

func createTerraformProvider(ctx context.Context) *TerraformProvider {
	projectDir := "../../../../test/samples/resourcegroupterraform"
	options := Options{
		Module: "main",
	}

	env := environment.Environment{Values: make(map[string]string)}
	env.SetLocation("westus2")
	env.SetEnvName("test-env")

	return NewTerraformProvider(ctx, &env, projectDir, options)
}

func prepareGenericMocks(execUtil *execmock.MockCommandRunner) {

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, "terraform version")
	}).Respond(executil.RunResult{
		Stdout: `{"terraform_version": "1.1.7"}`,
		Stderr: "",
	})
}

func preparePlanningMocks(execUtil *execmock.MockCommandRunner) {
	modulePath := "..\\..\\..\\..\\test\\samples\\resourcegroupterraform\\infra"

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s init", modulePath))
	}).Respond(executil.RunResult{
		Stdout: string("Terraform has been successfully initialized!"),
		Stderr: "",
	})

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s validate", modulePath))
	}).Respond(executil.RunResult{
		Stdout: string("Success! The configuration is valid."),
		Stderr: "",
	})

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s plan", modulePath))
	}).Respond(executil.RunResult{
		Stdout: string("To perform exactly these actions, run the following command to apply:terraform apply"),
		Stderr: "",
	})
}

func prepareDeployMocks(execUtil *execmock.MockCommandRunner) {
	modulePath := "..\\..\\..\\..\\test\\samples\\resourcegroupterraform\\infra"

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s validate", modulePath))
	}).Respond(executil.RunResult{
		Stdout: string("Success! The configuration is valid."),
		Stderr: "",
	})

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s apply", modulePath))
	}).Respond(executil.RunResult{
		Stdout: string(""),
		Stderr: "",
	})

	output := fmt.Sprintf("{\"AZURE_LOCATION\": {\"sensitive\": false,\"type\": \"string\",\"value\": \"westus2\"},\"RG_NAME\":{\"sensitive\": false,\"type\": \"string\",\"value\": \"rg-test-env\"}}")
	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s output", modulePath))
	}).Respond(executil.RunResult{
		Stdout: output,
		Stderr: "",
	})
}

func prepareDestroyMocks(execUtil *execmock.MockCommandRunner) {
	modulePath := "..\\..\\..\\..\\test\\samples\\resourcegroupterraform\\infra"

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s init", modulePath))
	}).Respond(executil.RunResult{
		Stdout: string("Terraform has been successfully initialized!"),
		Stderr: "",
	})

	output := fmt.Sprintf("{\"AZURE_LOCATION\": {\"sensitive\": false,\"type\": \"string\",\"value\": \"westus2\"},\"RG_NAME\":{\"sensitive\": false,\"type\": \"string\",\"value\": \"rg-test-env\"}}")
	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s output", modulePath))
	}).Respond(executil.RunResult{
		Stdout: output,
		Stderr: "",
	})

	execUtil.When(func(args executil.RunArgs, command string) bool {
		return strings.Contains(command, fmt.Sprintf("terraform -chdir=%s destroy", modulePath))
	}).Respond(executil.RunResult{
		Stdout: string(""),
		Stderr: "",
	})

}
