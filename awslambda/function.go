package awslambda

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	intranetzip "github.com/src-bin/substrate/cmd/substrate/intranet-zip"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	ResourceConflictException = "ResourceConflictException"

	handler = "bootstrap"
	runtime = types.RuntimeProvidedal2
)

func EnsureFunction(
	ctx context.Context,
	cfg *awscfg.Config,
	name, roleARN string,
	environment map[string]string,
	code []byte,
) (functionARN string, err error) {
	ui.Spinf("finding or creating the %s Lambda function", name)
	functionARN, err = createFunction(ctx, cfg, name, roleARN, environment, code)
	if awsutil.ErrorCodeIs(err, ResourceConflictException) {
		ui.Stop("already exists")
		ui.Spinf("updating the %s Lambda function's configuration", name)
		for range awsutil.JitteredExponentialBackoff(time.Second, 10*time.Second) {
			functionARN, err = updateFunctionConfiguration(ctx, cfg, name, roleARN, environment)
			if !awsutil.ErrorCodeIs(err, ResourceConflictException) {
				break
			} else {
				ui.Debug(err)
				// time.Sleep(1e9) // TODO exponential backoff
			}
		}
		ui.StopErr(err)
		if err != nil {
			return
		}
		ui.Spinf("updating the %s Lambda function's code", name)
		err = updateFunctionCode(ctx, cfg, name, code)
	}
	ui.StopErr(err)
	return
}

func createFunction(
	ctx context.Context,
	cfg *awscfg.Config,
	name, roleARN string,
	environment map[string]string,
	code []byte,
) (functionARN string, err error) {
	var out *lambda.CreateFunctionOutput
	out, err = cfg.Lambda().CreateFunction(ctx, &lambda.CreateFunctionInput{
		Architectures: []types.Architecture{types.ArchitectureArm64},
		Code:          &types.FunctionCode{ZipFile: intranetzip.SubstrateIntranetZip},
		Environment:   &types.Environment{Variables: environment},
		FunctionName:  aws.String(name),
		Handler:       aws.String(handler),
		PackageType:   types.PackageTypeZip,
		Role:          aws.String(roleARN),
		Runtime:       runtime,
		Tags: tagging.Map{
			tagging.Manager:          tagging.Substrate,
			tagging.SubstrateVersion: version.Version,
		},
	})
	if err == nil {
		functionARN = aws.ToString(out.FunctionArn)
	}
	return
}

func updateFunctionCode(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	code []byte,
) error {
	_, err := cfg.Lambda().UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		Architectures: []types.Architecture{types.ArchitectureArm64},
		Publish:       true,
		ZipFile:       intranetzip.SubstrateIntranetZip,
		FunctionName:  aws.String(name),
	})
	return err
}

func updateFunctionConfiguration(
	ctx context.Context,
	cfg *awscfg.Config,
	name, roleARN string,
	environment map[string]string,
) (functionARN string, err error) {
	var out *lambda.UpdateFunctionConfigurationOutput
	out, err = cfg.Lambda().UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		Environment:  &types.Environment{Variables: environment},
		FunctionName: aws.String(name),
		Handler:      aws.String(handler),
		Role:         aws.String(roleARN),
		Runtime:      runtime,
	})
	if err == nil {
		functionARN = aws.ToString(out.FunctionArn)
	}
	return
}
