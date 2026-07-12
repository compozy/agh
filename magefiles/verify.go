//go:build mage

package main

func Verify() error {
	release := acquireVerifyLock()
	defer release()
	return runMageSteps(verifySteps())
}

func verifySteps() []mageStep {
	return []mageStep{
		{name: "CodegenCheck", run: CodegenCheck},
		{name: "MarkCodegenChecked", run: markCodegenChecked},
		{name: "InstallerCheck", run: InstallerCheck},
		{name: "BunLint", run: BunLint},
		{name: "BunTypecheck", run: BunTypecheck},
		{name: "BunTest", run: BunTest},
		{name: "WebBuild", run: WebBuild},
		{name: "Fmt", run: Fmt},
		{name: "GoLint", run: goLint},
		{name: "Test", run: Test},
		{name: "buildGo", run: buildGo},
		{name: "Boundaries", run: Boundaries},
	}
}

func runMageSteps(steps []mageStep) error {
	for _, step := range steps {
		if err := step.run(); err != nil {
			return err
		}
	}
	return nil
}
